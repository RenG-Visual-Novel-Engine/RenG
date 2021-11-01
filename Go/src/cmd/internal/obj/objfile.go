// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Writing Go object files.

package obj

import (
	"bytes"
	"cmd/internal/bio"
	"cmd/internal/goobj"
	"cmd/internal/objabi"
	"cmd/internal/sys"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Entry point of writing new object file.
func WriteObjFile(ctxt *Link, b *bio.Writer) {

	debugAsmEmit(ctxt)

	genFuncInfoSyms(ctxt)

	w := writer{
		Writer:  goobj.NewWriter(b),
		ctxt:    ctxt,
		pkgpath: objabi.PathToPrefix(ctxt.Pkgpath),
	}

	start := b.Offset()
	w.init()

	// Header
	// We just reserve the space. We'll fill in the offsets later.
	flags := uint32(0)
	if ctxt.Flag_shared {
		flags |= goobj.ObjFlagShared
	}
	if w.pkgpath == "" {
		flags |= goobj.ObjFlagNeedNameExpansion
	}
	if ctxt.IsAsm {
		flags |= goobj.ObjFlagFromAssembly
	}
	h := goobj.Header{
		Magic:       goobj.Magic,
		Fingerprint: ctxt.Fingerprint,
		Flags:       flags,
	}
	h.Write(w.Writer)

	// String table
	w.StringTable()

	// Autolib
	h.Offsets[goobj.BlkAutolib] = w.Offset()
	for i := range ctxt.Imports {
		ctxt.Imports[i].Write(w.Writer)
	}

	// Package references
	h.Offsets[goobj.BlkPkgIdx] = w.Offset()
	for _, pkg := range w.pkglist {
		w.StringRef(pkg)
	}

	// File table (for DWARF and pcln generation).
	h.Offsets[goobj.BlkFile] = w.Offset()
	for _, f := range ctxt.PosTable.FileTable() {
		w.StringRef(filepath.ToSlash(f))
	}

	// Symbol definitions
	h.Offsets[goobj.BlkSymdef] = w.Offset()
	for _, s := range ctxt.defs {
		w.Sym(s)
	}

	// Short hashed symbol definitions
	h.Offsets[goobj.BlkHashed64def] = w.Offset()
	for _, s := range ctxt.hashed64defs {
		w.Sym(s)
	}

	// Hashed symbol definitions
	h.Offsets[goobj.BlkHasheddef] = w.Offset()
	for _, s := range ctxt.hasheddefs {
		w.Sym(s)
	}

	// Non-pkg symbol definitions
	h.Offsets[goobj.BlkNonpkgdef] = w.Offset()
	for _, s := range ctxt.nonpkgdefs {
		w.Sym(s)
	}

	// Non-pkg symbol references
	h.Offsets[goobj.BlkNonpkgref] = w.Offset()
	for _, s := range ctxt.nonpkgrefs {
		w.Sym(s)
	}

	// Referenced package symbol flags
	h.Offsets[goobj.BlkRefFlags] = w.Offset()
	w.refFlags()

	// Hashes
	h.Offsets[goobj.BlkHash64] = w.Offset()
	for _, s := range ctxt.hashed64defs {
		w.Hash64(s)
	}
	h.Offsets[goobj.BlkHash] = w.Offset()
	for _, s := range ctxt.hasheddefs {
		w.Hash(s)
	}
	// TODO: hashedrefs unused/unsupported for now

	// Reloc indexes
	h.Offsets[goobj.BlkRelocIdx] = w.Offset()
	nreloc := uint32(0)
	lists := [][]*LSym{ctxt.defs, ctxt.hashed64defs, ctxt.hasheddefs, ctxt.nonpkgdefs}
	for _, list := range lists {
		for _, s := range list {
			w.Uint32(nreloc)
			nreloc += uint32(len(s.R))
		}
	}
	w.Uint32(nreloc)

	// Symbol Info indexes
	h.Offsets[goobj.BlkAuxIdx] = w.Offset()
	naux := uint32(0)
	for _, list := range lists {
		for _, s := range list {
			w.Uint32(naux)
			naux += uint32(nAuxSym(s))
		}
	}
	w.Uint32(naux)

	// Data indexes
	h.Offsets[goobj.BlkDataIdx] = w.Offset()
	dataOff := int64(0)
	for _, list := range lists {
		for _, s := range list {
			w.Uint32(uint32(dataOff))
			dataOff += int64(len(s.P))
			if file := s.File(); file != nil {
				dataOff += int64(file.Size)
			}
		}
	}
	if int64(uint32(dataOff)) != dataOff {
		log.Fatalf("data too large")
	}
	w.Uint32(uint32(dataOff))

	// Relocs
	h.Offsets[goobj.BlkReloc] = w.Offset()
	for _, list := range lists {
		for _, s := range list {
			for i := range s.R {
				w.Reloc(&s.R[i])
			}
		}
	}

	// Aux symbol info
	h.Offsets[goobj.BlkAux] = w.Offset()
	for _, list := range lists {
		for _, s := range list {
			w.Aux(s)
		}
	}

	// Data
	h.Offsets[goobj.BlkData] = w.Offset()
	for _, list := range lists {
		for _, s := range list {
			w.Bytes(s.P)
			if file := s.File(); file != nil {
				w.writeFile(ctxt, file)
			}
		}
	}

	// Pcdata
	h.Offsets[goobj.BlkPcdata] = w.Offset()
	for _, s := range ctxt.Text { // iteration order must match genFuncInfoSyms
		// Because of the phase order, it's possible that we try to write an invalid
		// object file, and the Pcln variables haven't been filled in. As such, we
		// need to check that Pcsp exists, and assume the other pcln variables exist
		// as well. Tests like test/fixedbugs/issue22200.go demonstrate this issue.
		if fn := s.Func(); fn != nil && fn.Pcln.Pcsp != nil {
			pc := &fn.Pcln
			w.Bytes(pc.Pcsp.P)
			w.Bytes(pc.Pcfile.P)
			w.Bytes(pc.Pcline.P)
			w.Bytes(pc.Pcinline.P)
			for i := range pc.Pcdata {
				w.Bytes(pc.Pcdata[i].P)
			}
		}
	}

	// Blocks used only by tools (objdump, nm).

	// Referenced symbol names from other packages
	h.Offsets[goobj.BlkRefName] = w.Offset()
	w.refNames()

	h.Offsets[goobj.BlkEnd] = w.Offset()

	// Fix up block offsets in the header
	end := start + int64(w.Offset())
	b.MustSeek(start, 0)
	h.Write(w.Writer)
	b.MustSeek(end, 0)
}

type writer struct {
	*goobj.Writer
	filebuf []byte
	ctxt    *Link
	pkgpath string   // the package import path (escaped), "" if unknown
	pkglist []string // list of packages referenced, indexed by ctxt.pkgIdx
}

// prepare package index list
func (w *writer) init() {
	w.pkglist = make([]string, len(w.ctxt.pkgIdx)+1)
	w.pkglist[0] = "" // dummy invalid package for index 0
	for pkg, i := range w.ctxt.pkgIdx {
		w.pkglist[i] = pkg
	}
}

func (w *writer) writeFile(ctxt *Link, file *FileInfo) {
	f, err := os.Open(file.Name)
	if err != nil {
		ctxt.Diag("%v", err)
		return
	}
	defer f.Close()
	if w.filebuf == nil {
		w.filebuf = make([]byte, 1024)
	}
	buf := w.filebuf
	written := int64(0)
	for {
		n, err := f.Read(buf)
		w.Bytes(buf[:n])
		written += int64(n)
		if err == io.EOF {
			break
		}
		if err != nil {
			ctxt.Diag("%v", err)
			return
		}
	}
	if written != file.Size {
		ctxt.Diag("copy %s: unexpected length %d != %d", file.Name, written, file.Size)
	}
}

func (w *writer) StringTable() {
	w.AddString("")
	for _, p := range w.ctxt.Imports {
		w.AddString(p.Pkg)
	}
	for _, pkg := range w.pkglist {
		w.AddString(pkg)
	}
	w.ctxt.traverseSyms(traverseAll, func(s *LSym) {
		// TODO: this includes references of indexed symbols from other packages,
		// for which the linker doesn't need the name. Consider moving them to
		// a separate block (for tools only).
		if w.pkgpath != "" {
			s.Name = strings.Replace(s.Name, "\"\".", w.pkgpath+".", -1)
		}
		// Don't put names of builtins into the string table (to save
		// space).
		if s.PkgIdx == goobj.PkgIdxBuiltin {
			return
		}
		w.AddString(s.Name)
	})

	// All filenames are in the postable.
	for _, f := range w.ctxt.PosTable.FileTable() {
		w.AddString(filepath.ToSlash(f))
	}
}

// cutoff is the maximum data section size permitted by the linker
// (see issue #9862).
const cutoff = int64(2e9) // 2 GB (or so; looks better in errors than 2^31)

func (w *writer) Sym(s *LSym) {
	abi := uint16(s.ABI())
	if s.Static() {
		abi = goobj.SymABIstatic
	}
	flag := uint8(0)
	if s.DuplicateOK() {
		flag |= goobj.SymFlagDupok
	}
	if s.Local() {
		flag |= goobj.SymFlagLocal
	}
	if s.MakeTypelink() {
		flag |= goobj.SymFlagTypelink
	}
	if s.Leaf() {
		flag |= goobj.SymFlagLeaf
	}
	if s.NoSplit() {
		flag |= goobj.SymFlagNoSplit
	}
	if s.ReflectMethod() {
		flag |= goobj.SymFlagReflectMethod
	}
	if strings.HasPrefix(s.Name, "type.") && s.Name[5] != '.' && s.Type == objabi.SRODATA {
		flag |= goobj.SymFlagGoType
	}
	flag2 := uint8(0)
	if s.UsedInIface() {
		flag2 |= goobj.SymFlagUsedInIface
	}
	if strings.HasPrefix(s.Name, "go.itab.") && s.Type == objabi.SRODATA {
		flag2 |= goobj.SymFlagItab
	}
	name := s.Name
	if strings.HasPrefix(name, "gofile..") {
		name = filepath.ToSlash(name)
	}
	var align uint32
	if fn := s.Func(); fn != nil {
		align = uint32(fn.Align)
	}
	if s.ContentAddressable() {
		// We generally assume data symbols are natually aligned,
		// except for strings. If we dedup a string symbol and a
		// non-string symbol with the same content, we should keep
		// the largest alignment.
		// TODO: maybe the compiler could set the alignment for all
		// data symbols more carefully.
		if s.Size != 0 && !strings.HasPrefix(s.Name, "go.string.") {
			switch {
			case w.ctxt.Arch.PtrSize == 8 && s.Size%8 == 0:
				align = 8
			case s.Size%4 == 0:
				align = 4
			case s.Size%2 == 0:
				align = 2
			}
			// don't bother setting align to 1.
		}
	}
	if s.Size > cutoff {
		w.ctxt.Diag("%s: symbol too large (%d bytes > %d bytes)", s.Name, s.Size, cutoff)
	}
	var o goobj.Sym
	o.SetName(name, w.Writer)
	o.SetABI(abi)
	o.SetType(uint8(s.Type))
	o.SetFlag(flag)
	o.SetFlag2(flag2)
	o.SetSiz(uint32(s.Size))
	o.SetAlign(align)
	o.Write(w.Writer)
}

func (w *writer) Hash64(s *LSym) {
	if !s.ContentAddressable() || len(s.R) != 0 {
		panic("Hash of non-content-addressable symbol")
	}
	b := contentHash64(s)
	w.Bytes(b[:])
}

func (w *writer) Hash(s *LSym) {
	if !s.ContentAddressable() {
		panic("Hash of non-content-addressable symbol")
	}
	b := w.contentHash(s)
	w.Bytes(b[:])
}

func contentHash64(s *LSym) goobj.Hash64Type {
	var b goobj.Hash64Type
	copy(b[:], s.P)
	return b
}

// Compute the content hash for a content-addressable symbol.
// We build a content hash based on its content and relocations.
// Depending on the category of the referenced symbol, we choose
// different hash algorithms such that the hash is globally
// consistent.
// - For referenced content-addressable symbol, its content hash
//   is globally consistent.
// - For package symbol and builtin symbol, its local index is
//   globally consistent.
// - For non-package symbol, its fully-expanded name is globally
//   consistent. For now, we require we know the current package
//   path so we can always expand symbol names. (Otherwise,
//   symbols with relocations are not considered hashable.)
//
// For now, we assume there is no circular dependencies among
// hashed symbols.
func (w *writer) contentHash(s *LSym) goobj.HashType {
	h := sha1.New()
	var tmp [14]byte

	// Include the size of the symbol in the hash.
	// This preserves the length of symbols, preventing the following two symbols
	// from hashing the same:
	//
	//    [2]int{1,2} ≠ [10]int{1,2,0,0,0...}
	//
	// In this case, if the smaller symbol is alive, the larger is not kept unless
	// needed.
	binary.LittleEndian.PutUint64(tmp[:8], uint64(s.Size))
	h.Write(tmp[:8])

	// Don't dedup type symbols with others, as they are in a different
	// section.
	if strings.HasPrefix(s.Name, "type.") {
		h.Write([]byte{'T'})
	} else {
		h.Write([]byte{0})
	}
	// The compiler trims trailing zeros _sometimes_. We just do
	// it always.
	h.Write(bytes.TrimRight(s.P, "\x00"))
	for i := range s.R {
		r := &s.R[i]
		binary.LittleEndian.PutUint32(tmp[:4], uint32(r.Off))
		tmp[4] = r.Siz
		tmp[5] = uint8(r.Type)
		binary.LittleEndian.PutUint64(tmp[6:14], uint64(r.Add))
		h.Write(tmp[:])
		rs := r.Sym
		switch rs.PkgIdx {
		case goobj.PkgIdxHashed64:
			h.Write([]byte{0})
			t := contentHash64(rs)
			h.Write(t[:])
		case goobj.PkgIdxHashed:
			h.Write([]byte{1})
			t := w.contentHash(rs)
			h.Write(t[:])
		case goobj.PkgIdxNone:
			h.Write([]byte{2})
			io.WriteString(h, rs.Name) // name is already expanded at this point
		case goobj.PkgIdxBuiltin:
			h.Write([]byte{3})
			binary.LittleEndian.PutUint32(tmp[:4], uint32(rs.SymIdx))
			h.Write(tmp[:4])
		case goobj.PkgIdxSelf:
			io.WriteString(h, w.pkgpath)
			binary.LittleEndian.PutUint32(tmp[:4], uint32(rs.SymIdx))
			h.Write(tmp[:4])
		default:
			io.WriteString(h, rs.Pkg)
			binary.LittleEndian.PutUint32(tmp[:4], uint32(rs.SymIdx))
			h.Write(tmp[:4])
		}
	}
	var b goobj.HashType
	copy(b[:], h.Sum(nil))
	return b
}

func makeSymRef(s *LSym) goobj.SymRef {
	if s == nil {
		return goobj.SymRef{}
	}
	if s.PkgIdx == 0 || !s.Indexed() {
		fmt.Printf("unindexed symbol reference: %v\n", s)
		panic("unindexed symbol reference")
	}
	return goobj.SymRef{PkgIdx: uint32(s.PkgIdx), SymIdx: uint32(s.SymIdx)}
}

func (w *writer) Reloc(r *Reloc) {
	var o goobj.Reloc
	o.SetOff(r.Off)
	o.SetSiz(r.Siz)
	o.SetType(uint16(r.Type))
	o.SetAdd(r.Add)
	o.SetSym(makeSymRef(r.Sym))
	o.Write(w.Writer)
}

func (w *writer) aux1(typ uint8, rs *LSym) {
	var o goobj.Aux
	o.SetType(typ)
	o.SetSym(makeSymRef(rs))
	o.Write(w.Writer)
}

func (w *writer) Aux(s *LSym) {
	if s.Gotype != nil {
		w.aux1(goobj.AuxGotype, s.Gotype)
	}
	if fn := s.Func(); fn != nil {
		w.aux1(goobj.AuxFuncInfo, fn.FuncInfoSym)

		for _, d := range fn.Pcln.Funcdata {
			w.aux1(goobj.AuxFuncdata, d)
		}

		if fn.dwarfInfoSym != nil && fn.dwarfInfoSym.Size != 0 {
			w.aux1(goobj.AuxDwarfInfo, fn.dwarfInfoSym)
		}
		if fn.dwarfLocSym != nil && fn.dwarfLocSym.Size != 0 {
			w.aux1(goobj.AuxDwarfLoc, fn.dwarfLocSym)
		}
		if fn.dwarfRangesSym != nil && fn.dwarfRangesSym.Size != 0 {
			w.aux1(goobj.AuxDwarfRanges, fn.dwarfRangesSym)
		}
		if fn.dwarfDebugLinesSym != nil && fn.dwarfDebugLinesSym.Size != 0 {
			w.aux1(goobj.AuxDwarfLines, fn.dwarfDebugLinesSym)
		}
		if fn.Pcln.Pcsp != nil && fn.Pcln.Pcsp.Size != 0 {
			w.aux1(goobj.AuxPcsp, fn.Pcln.Pcsp)
		}
		if fn.Pcln.Pcfile != nil && fn.Pcln.Pcfile.Size != 0 {
			w.aux1(goobj.AuxPcfile, fn.Pcln.Pcfile)
		}
		if fn.Pcln.Pcline != nil && fn.Pcln.Pcline.Size != 0 {
			w.aux1(goobj.AuxPcline, fn.Pcln.Pcline)
		}
		if fn.Pcln.Pcinline != nil && fn.Pcln.Pcinline.Size != 0 {
			w.aux1(goobj.AuxPcinline, fn.Pcln.Pcinline)
		}
		for _, pcSym := range fn.Pcln.Pcdata {
			w.aux1(goobj.AuxPcdata, pcSym)
		}

	}
}

// Emits flags of referenced indexed symbols.
func (w *writer) refFlags() {
	seen := make(map[*LSym]bool)
	w.ctxt.traverseSyms(traverseRefs, func(rs *LSym) { // only traverse refs, not auxs, as tools don't need auxs
		switch rs.PkgIdx {
		case goobj.PkgIdxNone, goobj.PkgIdxHashed64, goobj.PkgIdxHashed, goobj.PkgIdxBuiltin, goobj.PkgIdxSelf: // not an external indexed reference
			return
		case goobj.PkgIdxInvalid:
			panic("unindexed symbol reference")
		}
		if seen[rs] {
			return
		}
		seen[rs] = true
		symref := makeSymRef(rs)
		flag2 := uint8(0)
		if rs.UsedInIface() {
			flag2 |= goobj.SymFlagUsedInIface
		}
		if flag2 == 0 {
			return // no need to write zero flags
		}
		var o goobj.RefFlags
		o.SetSym(symref)
		o.SetFlag2(flag2)
		o.Write(w.Writer)
	})
}

// Emits names of referenced indexed symbols, used by tools (objdump, nm)
// only.
func (w *writer) refNames() {
	seen := make(map[*LSym]bool)
	w.ctxt.traverseSyms(traverseRefs, func(rs *LSym) { // only traverse refs, not auxs, as tools don't need auxs
		switch rs.PkgIdx {
		case goobj.PkgIdxNone, goobj.PkgIdxHashed64, goobj.PkgIdxHashed, goobj.PkgIdxBuiltin, goobj.PkgIdxSelf: // not an external indexed reference
			return
		case goobj.PkgIdxInvalid:
			panic("unindexed symbol reference")
		}
		if seen[rs] {
			return
		}
		seen[rs] = true
		symref := makeSymRef(rs)
		var o goobj.RefName
		o.SetSym(symref)
		o.SetName(rs.Name, w.Writer)
		o.Write(w.Writer)
	})
	// TODO: output in sorted order?
	// Currently tools (cmd/internal/goobj package) doesn't use mmap,
	// and it just read it into a map in memory upfront. If it uses
	// mmap, if the output is sorted, it probably could avoid reading
	// into memory and just do lookups in the mmap'd object file.
}

// return the number of aux symbols s have.
func nAuxSym(s *LSym) int {
	n := 0
	if s.Gotype != nil {
		n++
	}
	if fn := s.Func(); fn != nil {
		// FuncInfo is an aux symbol, each Funcdata is an aux symbol
		n += 1 + len(fn.Pcln.Funcdata)
		if fn.dwarfInfoSym != nil && fn.dwarfInfoSym.Size != 0 {
			n++
		}
		if fn.dwarfLocSym != nil && fn.dwarfLocSym.Size != 0 {
			n++
		}
		if fn.dwarfRangesSym != nil && fn.dwarfRangesSym.Size != 0 {
			n++
		}
		if fn.dwarfDebugLinesSym != nil && fn.dwarfDebugLinesSym.Size != 0 {
			n++
		}
		if fn.Pcln.Pcsp != nil && fn.Pcln.Pcsp.Size != 0 {
			n++
		}
		if fn.Pcln.Pcfile != nil && fn.Pcln.Pcfile.Size != 0 {
			n++
		}
		if fn.Pcln.Pcline != nil && fn.Pcln.Pcline.Size != 0 {
			n++
		}
		if fn.Pcln.Pcinline != nil && fn.Pcln.Pcinline.Size != 0 {
			n++
		}
		n += len(fn.Pcln.Pcdata)
	}
	return n
}

// generate symbols for FuncInfo.
func genFuncInfoSyms(ctxt *Link) {
	infosyms := make([]*LSym, 0, len(ctxt.Text))
	hashedsyms := make([]*LSym, 0, 4*len(ctxt.Text))
	preparePcSym := func(s *LSym) *LSym {
		if s == nil {
			return s
		}
		s.PkgIdx = goobj.PkgIdxHashed
		s.SymIdx = int32(len(hashedsyms) + len(ctxt.hasheddefs))
		s.Set(AttrIndexed, true)
		hashedsyms = append(hashedsyms, s)
		return s
	}
	var b bytes.Buffer
	symidx := int32(len(ctxt.defs))
	for _, s := range ctxt.Text {
		fn := s.Func()
		if fn == nil {
			continue
		}
		o := goobj.FuncInfo{
			Args:     uint32(fn.Args),
			Locals:   uint32(fn.Locals),
			FuncID:   fn.FuncID,
			FuncFlag: fn.FuncFlag,
		}
		pc := &fn.Pcln
		o.Pcsp = makeSymRef(preparePcSym(pc.Pcsp))
		o.Pcfile = makeSymRef(preparePcSym(pc.Pcfile))
		o.Pcline = makeSymRef(preparePcSym(pc.Pcline))
		o.Pcinline = makeSymRef(preparePcSym(pc.Pcinline))
		o.Pcdata = make([]goobj.SymRef, len(pc.Pcdata))
		for i, pcSym := range pc.Pcdata {
			o.Pcdata[i] = makeSymRef(preparePcSym(pcSym))
		}
		o.Funcdataoff = make([]uint32, len(pc.Funcdataoff))
		for i, x := range pc.Funcdataoff {
			o.Funcdataoff[i] = uint32(x)
		}
		i := 0
		o.File = make([]goobj.CUFileIndex, len(pc.UsedFiles))
		for f := range pc.UsedFiles {
			o.File[i] = f
			i++
		}
		sort.Slice(o.File, func(i, j int) bool { return o.File[i] < o.File[j] })
		o.InlTree = make([]goobj.InlTreeNode, len(pc.InlTree.nodes))
		for i, inl := range pc.InlTree.nodes {
			f, l := getFileIndexAndLine(ctxt, inl.Pos)
			o.InlTree[i] = goobj.InlTreeNode{
				Parent:   int32(inl.Parent),
				File:     goobj.CUFileIndex(f),
				Line:     l,
				Func:     makeSymRef(inl.Func),
				ParentPC: inl.ParentPC,
			}
		}

		o.Write(&b)
		isym := &LSym{
			Type:   objabi.SDATA, // for now, I don't think it matters
			PkgIdx: goobj.PkgIdxSelf,
			SymIdx: symidx,
			P:      append([]byte(nil), b.Bytes()...),
		}
		isym.Set(AttrIndexed, true)
		symidx++
		infosyms = append(infosyms, isym)
		fn.FuncInfoSym = isym
		b.Reset()

		dwsyms := []*LSym{fn.dwarfRangesSym, fn.dwarfLocSym, fn.dwarfDebugLinesSym, fn.dwarfInfoSym}
		for _, s := range dwsyms {
			if s == nil || s.Size == 0 {
				continue
			}
			s.PkgIdx = goobj.PkgIdxSelf
			s.SymIdx = symidx
			s.Set(AttrIndexed, true)
			symidx++
			infosyms = append(infosyms, s)
		}
	}
	ctxt.defs = append(ctxt.defs, infosyms...)
	ctxt.hasheddefs = append(ctxt.hasheddefs, hashedsyms...)
}

func writeAuxSymDebug(ctxt *Link, par *LSym, aux *LSym) {
	// Most aux symbols (ex: funcdata) are not interesting--
	// pick out just the DWARF ones for now.
	if aux.Type != objabi.SDWARFLOC &&
		aux.Type != objabi.SDWARFFCN &&
		aux.Type != objabi.SDWARFABSFCN &&
		aux.Type != objabi.SDWARFLINES &&
		aux.Type != objabi.SDWARFRANGE {
		return
	}
	ctxt.writeSymDebugNamed(aux, "aux for "+par.Name)
}

func debugAsmEmit(ctxt *Link) {
	if ctxt.Debugasm > 0 {
		ctxt.traverseSyms(traverseDefs, ctxt.writeSymDebug)
		if ctxt.Debugasm > 1 {
			fn := func(par *LSym, aux *LSym) {
				writeAuxSymDebug(ctxt, par, aux)
			}
			ctxt.traverseAuxSyms(traverseAux, fn)
		}
	}
}

func (ctxt *Link) writeSymDebug(s *LSym) {
	ctxt.writeSymDebugNamed(s, s.Name)
}

func (ctxt *Link) writeSymDebugNamed(s *LSym, name string) {
	ver := ""
	if ctxt.Debugasm > 1 {
		ver = fmt.Sprintf("<%d>", s.ABI())
	}
	fmt.Fprintf(ctxt.Bso, "%s%s ", name, ver)
	if s.Type != 0 {
		fmt.Fprintf(ctxt.Bso, "%v ", s.Type)
	}
	if s.Static() {
		fmt.Fprint(ctxt.Bso, "static ")
	}
	if s.DuplicateOK() {
		fmt.Fprintf(ctxt.Bso, "dupok ")
	}
	if s.CFunc() {
		fmt.Fprintf(ctxt.Bso, "cfunc ")
	}
	if s.NoSplit() {
		fmt.Fprintf(ctxt.Bso, "nosplit ")
	}
	if s.Func() != nil && s.Func().FuncFlag&objabi.FuncFlag_TOPFRAME != 0 {
		fmt.Fprintf(ctxt.Bso, "topframe ")
	}
	fmt.Fprintf(ctxt.Bso, "size=%d", s.Size)
	if s.Type == objabi.STEXT {
		fn := s.Func()
		fmt.Fprintf(ctxt.Bso, " args=%#x locals=%#x funcid=%#x", uint64(fn.Args), uint64(fn.Locals), uint64(fn.FuncID))
		if s.Leaf() {
			fmt.Fprintf(ctxt.Bso, " leaf")
		}
	}
	fmt.Fprintf(ctxt.Bso, "\n")
	if s.Type == objabi.STEXT {
		for p := s.Func().Text; p != nil; p = p.Link {
			fmt.Fprintf(ctxt.Bso, "\t%#04x ", uint(int(p.Pc)))
			if ctxt.Debugasm > 1 {
				io.WriteString(ctxt.Bso, p.String())
			} else {
				p.InnermostString(ctxt.Bso)
			}
			fmt.Fprintln(ctxt.Bso)
		}
	}
	for i := 0; i < len(s.P); i += 16 {
		fmt.Fprintf(ctxt.Bso, "\t%#04x", uint(i))
		j := i
		for ; j < i+16 && j < len(s.P); j++ {
			fmt.Fprintf(ctxt.Bso, " %02x", s.P[j])
		}
		for ; j < i+16; j++ {
			fmt.Fprintf(ctxt.Bso, "   ")
		}
		fmt.Fprintf(ctxt.Bso, "  ")
		for j = i; j < i+16 && j < len(s.P); j++ {
			c := int(s.P[j])
			b := byte('.')
			if ' ' <= c && c <= 0x7e {
				b = byte(c)
			}
			ctxt.Bso.WriteByte(b)
		}

		fmt.Fprintf(ctxt.Bso, "\n")
	}

	sort.Sort(relocByOff(s.R)) // generate stable output
	for _, r := range s.R {
		name := ""
		ver := ""
		if r.Sym != nil {
			name = r.Sym.Name
			if ctxt.Debugasm > 1 {
				ver = fmt.Sprintf("<%d>", r.Sym.ABI())
			}
		} else if r.Type == objabi.R_TLS_LE {
			name = "TLS"
		}
		if ctxt.Arch.InFamily(sys.ARM, sys.PPC64) {
			fmt.Fprintf(ctxt.Bso, "\trel %d+%d t=%d %s%s+%x\n", int(r.Off), r.Siz, r.Type, name, ver, uint64(r.Add))
		} else {
			fmt.Fprintf(ctxt.Bso, "\trel %d+%d t=%d %s%s+%d\n", int(r.Off), r.Siz, r.Type, name, ver, r.Add)
		}
	}
}

// relocByOff sorts relocations by their offsets.
type relocByOff []Reloc

func (x relocByOff) Len() int           { return len(x) }
func (x relocByOff) Less(i, j int) bool { return x[i].Off < x[j].Off }
func (x relocByOff) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
