// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"fmt"
	"sort"

	"cmd/compile/internal/base"
	"cmd/internal/src"
)

var PtrSize int

var RegSize int

// Slices in the runtime are represented by three components:
//
// type slice struct {
// 	ptr unsafe.Pointer
// 	len int
// 	cap int
// }
//
// Strings in the runtime are represented by two components:
//
// type string struct {
// 	ptr unsafe.Pointer
// 	len int
// }
//
// These variables are the offsets of fields and sizes of these structs.
var (
	SlicePtrOffset int64
	SliceLenOffset int64
	SliceCapOffset int64

	SliceSize  int64
	StringSize int64
)

var SkipSizeForTracing bool

// typePos returns the position associated with t.
// This is where t was declared or where it appeared as a type expression.
func typePos(t *Type) src.XPos {
	if pos := t.Pos(); pos.IsKnown() {
		return pos
	}
	base.Fatalf("bad type: %v", t)
	panic("unreachable")
}

// MaxWidth is the maximum size of a value on the target architecture.
var MaxWidth int64

// CalcSizeDisabled indicates whether it is safe
// to calculate Types' widths and alignments. See CalcSize.
var CalcSizeDisabled bool

// machine size and rounding alignment is dictated around
// the size of a pointer, set in betypeinit (see ../amd64/galign.go).
var defercalc int

func Rnd(o int64, r int64) int64 {
	if r < 1 || r > 8 || r&(r-1) != 0 {
		base.Fatalf("rnd %d", r)
	}
	return (o + r - 1) &^ (r - 1)
}

// expandiface computes the method set for interface type t by
// expanding embedded interfaces.
func expandiface(t *Type) {
	seen := make(map[*Sym]*Field)
	var methods []*Field

	addMethod := func(m *Field, explicit bool) {
		switch prev := seen[m.Sym]; {
		case prev == nil:
			seen[m.Sym] = m
		case AllowsGoVersion(t.Pkg(), 1, 14) && !explicit && Identical(m.Type, prev.Type):
			return
		default:
			base.ErrorfAt(m.Pos, "duplicate method %s", m.Sym.Name)
		}
		methods = append(methods, m)
	}

	for _, m := range t.Methods().Slice() {
		if m.Sym == nil {
			continue
		}

		CheckSize(m.Type)
		addMethod(m, true)
	}

	for _, m := range t.Methods().Slice() {
		if m.Sym != nil || m.Type == nil {
			continue
		}

		if !m.Type.IsInterface() {
			base.ErrorfAt(m.Pos, "interface contains embedded non-interface %v", m.Type)
			m.SetBroke(true)
			t.SetBroke(true)
			// Add to fields so that error messages
			// include the broken embedded type when
			// printing t.
			// TODO(mdempsky): Revisit this.
			methods = append(methods, m)
			continue
		}

		// Embedded interface: duplicate all methods
		// (including broken ones, if any) and add to t's
		// method set.
		for _, t1 := range m.Type.AllMethods().Slice() {
			// Use m.Pos rather than t1.Pos to preserve embedding position.
			f := NewField(m.Pos, t1.Sym, t1.Type)
			addMethod(f, false)
		}
	}

	sort.Sort(MethodsByName(methods))

	if int64(len(methods)) >= MaxWidth/int64(PtrSize) {
		base.ErrorfAt(typePos(t), "interface too large")
	}
	for i, m := range methods {
		m.Offset = int64(i) * int64(PtrSize)
	}

	t.SetAllMethods(methods)
}

func calcStructOffset(errtype *Type, t *Type, o int64, flag int) int64 {
	// flag is 0 (receiver), 1 (actual struct), or RegSize (in/out parameters)
	isStruct := flag == 1
	starto := o
	maxalign := int32(flag)
	if maxalign < 1 {
		maxalign = 1
	}
	lastzero := int64(0)
	for _, f := range t.Fields().Slice() {
		if f.Type == nil {
			// broken field, just skip it so that other valid fields
			// get a width.
			continue
		}

		CalcSize(f.Type)
		if int32(f.Type.Align) > maxalign {
			maxalign = int32(f.Type.Align)
		}
		if f.Type.Align > 0 {
			o = Rnd(o, int64(f.Type.Align))
		}
		if isStruct { // For receiver/args/results, do not set, it depends on ABI
			f.Offset = o
		}

		w := f.Type.Width
		if w < 0 {
			base.Fatalf("invalid width %d", f.Type.Width)
		}
		if w == 0 {
			lastzero = o
		}
		o += w
		maxwidth := MaxWidth
		// On 32-bit systems, reflect tables impose an additional constraint
		// that each field start offset must fit in 31 bits.
		if maxwidth < 1<<32 {
			maxwidth = 1<<31 - 1
		}
		if o >= maxwidth {
			base.ErrorfAt(typePos(errtype), "type %L too large", errtype)
			o = 8 // small but nonzero
		}
	}

	// For nonzero-sized structs which end in a zero-sized thing, we add
	// an extra byte of padding to the type. This padding ensures that
	// taking the address of the zero-sized thing can't manufacture a
	// pointer to the next object in the heap. See issue 9401.
	if flag == 1 && o > starto && o == lastzero {
		o++
	}

	// final width is rounded
	if flag != 0 {
		o = Rnd(o, int64(maxalign))
	}
	t.Align = uint8(maxalign)

	// type width only includes back to first field's offset
	t.Width = o - starto

	return o
}

// findTypeLoop searches for an invalid type declaration loop involving
// type t and reports whether one is found. If so, path contains the
// loop.
//
// path points to a slice used for tracking the sequence of types
// visited. Using a pointer to a slice allows the slice capacity to
// grow and limit reallocations.
func findTypeLoop(t *Type, path *[]*Type) bool {
	// We implement a simple DFS loop-finding algorithm. This
	// could be faster, but type cycles are rare.

	if t.Sym() != nil {
		// Declared type. Check for loops and otherwise
		// recurse on the type expression used in the type
		// declaration.

		// Type imported from package, so it can't be part of
		// a type loop (otherwise that package should have
		// failed to compile).
		if t.Sym().Pkg != LocalPkg {
			return false
		}

		for i, x := range *path {
			if x == t {
				*path = (*path)[i:]
				return true
			}
		}

		*path = append(*path, t)
		if findTypeLoop(t.Obj().(TypeObject).TypeDefn(), path) {
			return true
		}
		*path = (*path)[:len(*path)-1]
	} else {
		// Anonymous type. Recurse on contained types.

		switch t.Kind() {
		case TARRAY:
			if findTypeLoop(t.Elem(), path) {
				return true
			}
		case TSTRUCT:
			for _, f := range t.Fields().Slice() {
				if findTypeLoop(f.Type, path) {
					return true
				}
			}
		case TINTER:
			for _, m := range t.Methods().Slice() {
				if m.Type.IsInterface() { // embedded interface
					if findTypeLoop(m.Type, path) {
						return true
					}
				}
			}
		}
	}

	return false
}

func reportTypeLoop(t *Type) {
	if t.Broke() {
		return
	}

	var l []*Type
	if !findTypeLoop(t, &l) {
		base.Fatalf("failed to find type loop for: %v", t)
	}

	// Rotate loop so that the earliest type declaration is first.
	i := 0
	for j, t := range l[1:] {
		if typePos(t).Before(typePos(l[i])) {
			i = j + 1
		}
	}
	l = append(l[i:], l[:i]...)

	var msg bytes.Buffer
	fmt.Fprintf(&msg, "invalid recursive type %v\n", l[0])
	for _, t := range l {
		fmt.Fprintf(&msg, "\t%v: %v refers to\n", base.FmtPos(typePos(t)), t)
		t.SetBroke(true)
	}
	fmt.Fprintf(&msg, "\t%v: %v", base.FmtPos(typePos(l[0])), l[0])
	base.ErrorfAt(typePos(l[0]), msg.String())
}

// CalcSize calculates and stores the size and alignment for t.
// If CalcSizeDisabled is set, and the size/alignment
// have not already been calculated, it calls Fatal.
// This is used to prevent data races in the back end.
func CalcSize(t *Type) {
	// Calling CalcSize when typecheck tracing enabled is not safe.
	// See issue #33658.
	if base.EnableTrace && SkipSizeForTracing {
		return
	}
	if PtrSize == 0 {
		// Assume this is a test.
		return
	}

	if t == nil {
		return
	}

	if t.Width == -2 {
		reportTypeLoop(t)
		t.Width = 0
		t.Align = 1
		return
	}

	if t.WidthCalculated() {
		return
	}

	if CalcSizeDisabled {
		if t.Broke() {
			// break infinite recursion from Fatal call below
			return
		}
		t.SetBroke(true)
		base.Fatalf("width not calculated: %v", t)
	}

	// break infinite recursion if the broken recursive type
	// is referenced again
	if t.Broke() && t.Width == 0 {
		return
	}

	// defer CheckSize calls until after we're done
	DeferCheckSize()

	lno := base.Pos
	if pos := t.Pos(); pos.IsKnown() {
		base.Pos = pos
	}

	t.Width = -2
	t.Align = 0 // 0 means use t.Width, below

	et := t.Kind()
	switch et {
	case TFUNC, TCHAN, TMAP, TSTRING:
		break

	// SimType == 0 during bootstrap
	default:
		if SimType[t.Kind()] != 0 {
			et = SimType[t.Kind()]
		}
	}

	var w int64
	switch et {
	default:
		base.Fatalf("CalcSize: unknown type: %v", t)

	// compiler-specific stuff
	case TINT8, TUINT8, TBOOL:
		// bool is int8
		w = 1

	case TINT16, TUINT16:
		w = 2

	case TINT32, TUINT32, TFLOAT32:
		w = 4

	case TINT64, TUINT64, TFLOAT64:
		w = 8
		t.Align = uint8(RegSize)

	case TCOMPLEX64:
		w = 8
		t.Align = 4

	case TCOMPLEX128:
		w = 16
		t.Align = uint8(RegSize)

	case TPTR:
		w = int64(PtrSize)
		CheckSize(t.Elem())

	case TUNSAFEPTR:
		w = int64(PtrSize)

	case TINTER: // implemented as 2 pointers
		w = 2 * int64(PtrSize)
		t.Align = uint8(PtrSize)
		expandiface(t)

	case TCHAN: // implemented as pointer
		w = int64(PtrSize)

		CheckSize(t.Elem())

		// make fake type to check later to
		// trigger channel argument check.
		t1 := NewChanArgs(t)
		CheckSize(t1)

	case TCHANARGS:
		t1 := t.ChanArgs()
		CalcSize(t1) // just in case
		if t1.Elem().Width >= 1<<16 {
			base.ErrorfAt(typePos(t1), "channel element type too large (>64kB)")
		}
		w = 1 // anything will do

	case TMAP: // implemented as pointer
		w = int64(PtrSize)
		CheckSize(t.Elem())
		CheckSize(t.Key())

	case TFORW: // should have been filled in
		reportTypeLoop(t)
		w = 1 // anything will do

	case TANY:
		// not a real type; should be replaced before use.
		base.Fatalf("CalcSize any")

	case TSTRING:
		if StringSize == 0 {
			base.Fatalf("early CalcSize string")
		}
		w = StringSize
		t.Align = uint8(PtrSize)

	case TARRAY:
		if t.Elem() == nil {
			break
		}

		CalcSize(t.Elem())
		if t.Elem().Width != 0 {
			cap := (uint64(MaxWidth) - 1) / uint64(t.Elem().Width)
			if uint64(t.NumElem()) > cap {
				base.ErrorfAt(typePos(t), "type %L larger than address space", t)
			}
		}
		w = t.NumElem() * t.Elem().Width
		t.Align = t.Elem().Align

	case TSLICE:
		if t.Elem() == nil {
			break
		}
		w = SliceSize
		CheckSize(t.Elem())
		t.Align = uint8(PtrSize)

	case TSTRUCT:
		if t.IsFuncArgStruct() {
			base.Fatalf("CalcSize fn struct %v", t)
		}
		w = calcStructOffset(t, t, 0, 1)

	// make fake type to check later to
	// trigger function argument computation.
	case TFUNC:
		t1 := NewFuncArgs(t)
		CheckSize(t1)
		w = int64(PtrSize) // width of func type is pointer

	// function is 3 cated structures;
	// compute their widths as side-effect.
	case TFUNCARGS:
		t1 := t.FuncArgs()
		w = calcStructOffset(t1, t1.Recvs(), 0, 0)
		w = calcStructOffset(t1, t1.Params(), w, RegSize)
		w = calcStructOffset(t1, t1.Results(), w, RegSize)
		t1.Extra.(*Func).Argwid = w
		if w%int64(RegSize) != 0 {
			base.Warn("bad type %v %d\n", t1, w)
		}
		t.Align = 1

	case TTYPEPARAM:
		// TODO(danscales) - remove when we eliminate the need
		// to do CalcSize in noder2 (which shouldn't be needed in the noder)
		w = int64(PtrSize)
	}

	if PtrSize == 4 && w != int64(int32(w)) {
		base.ErrorfAt(typePos(t), "type %v too large", t)
	}

	t.Width = w
	if t.Align == 0 {
		if w == 0 || w > 8 || w&(w-1) != 0 {
			base.Fatalf("invalid alignment for %v", t)
		}
		t.Align = uint8(w)
	}

	base.Pos = lno

	ResumeCheckSize()
}

// CalcStructSize calculates the size of s,
// filling in s.Width and s.Align,
// even if size calculation is otherwise disabled.
func CalcStructSize(s *Type) {
	s.Width = calcStructOffset(s, s, 0, 1) // sets align
}

// when a type's width should be known, we call CheckSize
// to compute it.  during a declaration like
//
//	type T *struct { next T }
//
// it is necessary to defer the calculation of the struct width
// until after T has been initialized to be a pointer to that struct.
// similarly, during import processing structs may be used
// before their definition.  in those situations, calling
// DeferCheckSize() stops width calculations until
// ResumeCheckSize() is called, at which point all the
// CalcSizes that were deferred are executed.
// CalcSize should only be called when the type's size
// is needed immediately.  CheckSize makes sure the
// size is evaluated eventually.

var deferredTypeStack []*Type

func CheckSize(t *Type) {
	if t == nil {
		return
	}

	// function arg structs should not be checked
	// outside of the enclosing function.
	if t.IsFuncArgStruct() {
		base.Fatalf("CheckSize %v", t)
	}

	if defercalc == 0 {
		CalcSize(t)
		return
	}

	// if type has not yet been pushed on deferredTypeStack yet, do it now
	if !t.Deferwidth() {
		t.SetDeferwidth(true)
		deferredTypeStack = append(deferredTypeStack, t)
	}
}

func DeferCheckSize() {
	defercalc++
}

func ResumeCheckSize() {
	if defercalc == 1 {
		for len(deferredTypeStack) > 0 {
			t := deferredTypeStack[len(deferredTypeStack)-1]
			deferredTypeStack = deferredTypeStack[:len(deferredTypeStack)-1]
			t.SetDeferwidth(false)
			CalcSize(t)
		}
	}

	defercalc--
}

// PtrDataSize returns the length in bytes of the prefix of t
// containing pointer data. Anything after this offset is scalar data.
func PtrDataSize(t *Type) int64 {
	if !t.HasPointers() {
		return 0
	}

	switch t.Kind() {
	case TPTR,
		TUNSAFEPTR,
		TFUNC,
		TCHAN,
		TMAP:
		return int64(PtrSize)

	case TSTRING:
		// struct { byte *str; intgo len; }
		return int64(PtrSize)

	case TINTER:
		// struct { Itab *tab;	void *data; } or
		// struct { Type *type; void *data; }
		// Note: see comment in typebits.Set
		return 2 * int64(PtrSize)

	case TSLICE:
		// struct { byte *array; uintgo len; uintgo cap; }
		return int64(PtrSize)

	case TARRAY:
		// haspointers already eliminated t.NumElem() == 0.
		return (t.NumElem()-1)*t.Elem().Width + PtrDataSize(t.Elem())

	case TSTRUCT:
		// Find the last field that has pointers.
		var lastPtrField *Field
		fs := t.Fields().Slice()
		for i := len(fs) - 1; i >= 0; i-- {
			if fs[i].Type.HasPointers() {
				lastPtrField = fs[i]
				break
			}
		}
		return lastPtrField.Offset + PtrDataSize(lastPtrField.Type)

	default:
		base.Fatalf("PtrDataSize: unexpected type, %v", t)
		return 0
	}
}
