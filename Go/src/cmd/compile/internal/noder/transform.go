// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file contains transformation functions on nodes, which are the
// transformations that the typecheck package does that are distinct from the
// typechecking functionality. These transform functions are pared-down copies of
// the original typechecking functions, with all code removed that is related to:
//
//    - Detecting compile-time errors (already done by types2)
//    - Setting the actual type of existing nodes (already done based on
//      type info from types2)
//    - Dealing with untyped constants (which types2 has already resolved)
//
// Each of the transformation functions requires that node passed in has its type
// and typecheck flag set. If the transformation function replaces or adds new
// nodes, it will set the type and typecheck flag for those new nodes.

package noder

import (
	"cmd/compile/internal/base"
	"cmd/compile/internal/ir"
	"cmd/compile/internal/typecheck"
	"cmd/compile/internal/types"
	"fmt"
	"go/constant"
)

// Transformation functions for expressions

// transformAdd transforms an addition operation (currently just addition of
// strings). Corresponds to the "binary operators" case in typecheck.typecheck1.
func transformAdd(n *ir.BinaryExpr) ir.Node {
	assert(n.Type() != nil && n.Typecheck() == 1)
	l := n.X
	if l.Type().IsString() {
		var add *ir.AddStringExpr
		if l.Op() == ir.OADDSTR {
			add = l.(*ir.AddStringExpr)
			add.SetPos(n.Pos())
		} else {
			add = ir.NewAddStringExpr(n.Pos(), []ir.Node{l})
		}
		r := n.Y
		if r.Op() == ir.OADDSTR {
			r := r.(*ir.AddStringExpr)
			add.List.Append(r.List.Take()...)
		} else {
			add.List.Append(r)
		}
		typed(l.Type(), add)
		return add
	}
	return n
}

// Corresponds to typecheck.stringtoruneslit.
func stringtoruneslit(n *ir.ConvExpr) ir.Node {
	if n.X.Op() != ir.OLITERAL || n.X.Val().Kind() != constant.String {
		base.Fatalf("stringtoarraylit %v", n)
	}

	var list []ir.Node
	i := 0
	eltType := n.Type().Elem()
	for _, r := range ir.StringVal(n.X) {
		elt := ir.NewKeyExpr(base.Pos, ir.NewInt(int64(i)), ir.NewInt(int64(r)))
		// Change from untyped int to the actual element type determined
		// by types2.  No need to change elt.Key, since the array indexes
		// are just used for setting up the element ordering.
		elt.Value.SetType(eltType)
		list = append(list, elt)
		i++
	}

	nn := ir.NewCompLitExpr(base.Pos, ir.OCOMPLIT, ir.TypeNode(n.Type()), nil)
	nn.List = list
	typed(n.Type(), nn)
	// Need to transform the OCOMPLIT.
	return transformCompLit(nn)
}

// transformConv transforms an OCONV node as needed, based on the types involved,
// etc.  Corresponds to typecheck.tcConv.
func transformConv(n *ir.ConvExpr) ir.Node {
	t := n.X.Type()
	op, _ := typecheck.Convertop(n.X.Op() == ir.OLITERAL, t, n.Type())
	n.SetOp(op)
	switch n.Op() {
	case ir.OCONVNOP:
		if t.Kind() == n.Type().Kind() {
			switch t.Kind() {
			case types.TFLOAT32, types.TFLOAT64, types.TCOMPLEX64, types.TCOMPLEX128:
				// Floating point casts imply rounding and
				// so the conversion must be kept.
				n.SetOp(ir.OCONV)
			}
		}

	// Do not convert to []byte literal. See CL 125796.
	// Generated code and compiler memory footprint is better without it.
	case ir.OSTR2BYTES:
		// ok

	case ir.OSTR2RUNES:
		if n.X.Op() == ir.OLITERAL {
			return stringtoruneslit(n)
		}
	}
	return n
}

// transformConvCall transforms a conversion call. Corresponds to the OTYPE part of
// typecheck.tcCall.
func transformConvCall(n *ir.CallExpr) ir.Node {
	assert(n.Type() != nil && n.Typecheck() == 1)
	arg := n.Args[0]
	n1 := ir.NewConvExpr(n.Pos(), ir.OCONV, nil, arg)
	typed(n.X.Type(), n1)
	return transformConv(n1)
}

// transformCall transforms a normal function/method call. Corresponds to last half
// (non-conversion, non-builtin part) of typecheck.tcCall.
func transformCall(n *ir.CallExpr) {
	// n.Type() can be nil for calls with no return value
	assert(n.Typecheck() == 1)
	transformArgs(n)
	l := n.X
	t := l.Type()

	switch l.Op() {
	case ir.ODOTINTER:
		n.SetOp(ir.OCALLINTER)

	case ir.ODOTMETH:
		l := l.(*ir.SelectorExpr)
		n.SetOp(ir.OCALLMETH)

		tp := t.Recv().Type

		if l.X == nil || !types.Identical(l.X.Type(), tp) {
			base.Fatalf("method receiver")
		}

	default:
		n.SetOp(ir.OCALLFUNC)
	}

	typecheckaste(ir.OCALL, n.X, n.IsDDD, t.Params(), n.Args)
	if t.NumResults() == 1 {
		n.SetType(l.Type().Results().Field(0).Type)

		if n.Op() == ir.OCALLFUNC && n.X.Op() == ir.ONAME {
			if sym := n.X.(*ir.Name).Sym(); types.IsRuntimePkg(sym.Pkg) && sym.Name == "getg" {
				// Emit code for runtime.getg() directly instead of calling function.
				// Most such rewrites (for example the similar one for math.Sqrt) should be done in walk,
				// so that the ordering pass can make sure to preserve the semantics of the original code
				// (in particular, the exact time of the function call) by introducing temporaries.
				// In this case, we know getg() always returns the same result within a given function
				// and we want to avoid the temporaries, so we do the rewrite earlier than is typical.
				n.SetOp(ir.OGETG)
			}
		}
		return
	}
}

// transformCompare transforms a compare operation (currently just equals/not
// equals). Corresponds to the "comparison operators" case in
// typecheck.typecheck1, including tcArith.
func transformCompare(n *ir.BinaryExpr) {
	assert(n.Type() != nil && n.Typecheck() == 1)
	if (n.Op() == ir.OEQ || n.Op() == ir.ONE) && !types.Identical(n.X.Type(), n.Y.Type()) {
		// Comparison is okay as long as one side is assignable to the
		// other. The only allowed case where the conversion is not CONVNOP is
		// "concrete == interface". In that case, check comparability of
		// the concrete type. The conversion allocates, so only do it if
		// the concrete type is huge.
		l, r := n.X, n.Y
		lt, rt := l.Type(), r.Type()
		converted := false
		if rt.Kind() != types.TBLANK {
			aop, _ := typecheck.Assignop(lt, rt)
			if aop != ir.OXXX {
				types.CalcSize(lt)
				if rt.IsInterface() == lt.IsInterface() || lt.Width >= 1<<16 {
					l = ir.NewConvExpr(base.Pos, aop, rt, l)
					l.SetTypecheck(1)
				}

				converted = true
			}
		}

		if !converted && lt.Kind() != types.TBLANK {
			aop, _ := typecheck.Assignop(rt, lt)
			if aop != ir.OXXX {
				types.CalcSize(rt)
				if rt.IsInterface() == lt.IsInterface() || rt.Width >= 1<<16 {
					r = ir.NewConvExpr(base.Pos, aop, lt, r)
					r.SetTypecheck(1)
				}
			}
		}
		n.X, n.Y = l, r
	}
}

// Corresponds to typecheck.implicitstar.
func implicitstar(n ir.Node) ir.Node {
	// insert implicit * if needed for fixed array
	t := n.Type()
	if !t.IsPtr() {
		return n
	}
	t = t.Elem()
	if !t.IsArray() {
		return n
	}
	star := ir.NewStarExpr(base.Pos, n)
	star.SetImplicit(true)
	return typed(t, star)
}

// transformIndex transforms an index operation.  Corresponds to typecheck.tcIndex.
func transformIndex(n *ir.IndexExpr) {
	assert(n.Type() != nil && n.Typecheck() == 1)
	n.X = implicitstar(n.X)
	l := n.X
	t := l.Type()
	if t.Kind() == types.TMAP {
		n.Index = assignconvfn(n.Index, t.Key())
		n.SetOp(ir.OINDEXMAP)
		// Set type to just the map value, not (value, bool). This is
		// different from types2, but fits the later stages of the
		// compiler better.
		n.SetType(t.Elem())
		n.Assigned = false
	}
}

// transformSlice transforms a slice operation.  Corresponds to typecheck.tcSlice.
func transformSlice(n *ir.SliceExpr) {
	assert(n.Type() != nil && n.Typecheck() == 1)
	l := n.X
	if l.Type().IsArray() {
		addr := typecheck.NodAddr(n.X)
		addr.SetImplicit(true)
		typed(types.NewPtr(n.X.Type()), addr)
		n.X = addr
		l = addr
	}
	t := l.Type()
	if t.IsString() {
		n.SetOp(ir.OSLICESTR)
	} else if t.IsPtr() && t.Elem().IsArray() {
		if n.Op().IsSlice3() {
			n.SetOp(ir.OSLICE3ARR)
		} else {
			n.SetOp(ir.OSLICEARR)
		}
	}
}

// Transformation functions for statements

// Corresponds to typecheck.checkassign.
func transformCheckAssign(stmt ir.Node, n ir.Node) {
	if n.Op() == ir.OINDEXMAP {
		n := n.(*ir.IndexExpr)
		n.Assigned = true
		return
	}
}

// Corresponds to typecheck.assign.
func transformAssign(stmt ir.Node, lhs, rhs []ir.Node) {
	checkLHS := func(i int, typ *types.Type) {
		transformCheckAssign(stmt, lhs[i])
	}

	cr := len(rhs)
	if len(rhs) == 1 {
		if rtyp := rhs[0].Type(); rtyp != nil && rtyp.IsFuncArgStruct() {
			cr = rtyp.NumFields()
		}
	}

	// x, ok = y
assignOK:
	for len(lhs) == 2 && cr == 1 {
		stmt := stmt.(*ir.AssignListStmt)
		r := rhs[0]

		switch r.Op() {
		case ir.OINDEXMAP:
			stmt.SetOp(ir.OAS2MAPR)
		case ir.ORECV:
			stmt.SetOp(ir.OAS2RECV)
		case ir.ODOTTYPE:
			r := r.(*ir.TypeAssertExpr)
			stmt.SetOp(ir.OAS2DOTTYPE)
			r.SetOp(ir.ODOTTYPE2)
		default:
			break assignOK
		}
		checkLHS(0, r.Type())
		checkLHS(1, types.UntypedBool)
		return
	}

	if len(lhs) != cr {
		for i := range lhs {
			checkLHS(i, nil)
		}
		return
	}

	// x,y,z = f()
	if cr > len(rhs) {
		stmt := stmt.(*ir.AssignListStmt)
		stmt.SetOp(ir.OAS2FUNC)
		r := rhs[0].(*ir.CallExpr)
		r.Use = ir.CallUseList
		rtyp := r.Type()

		for i := range lhs {
			checkLHS(i, rtyp.Field(i).Type)
		}
		return
	}

	for i, r := range rhs {
		checkLHS(i, r.Type())
		if lhs[i].Type() != nil {
			rhs[i] = assignconvfn(r, lhs[i].Type())
		}
	}
}

// Corresponds to typecheck.typecheckargs.
func transformArgs(n ir.InitNode) {
	var list []ir.Node
	switch n := n.(type) {
	default:
		base.Fatalf("typecheckargs %+v", n.Op())
	case *ir.CallExpr:
		list = n.Args
		if n.IsDDD {
			return
		}
	case *ir.ReturnStmt:
		list = n.Results
	}
	if len(list) != 1 {
		return
	}

	t := list[0].Type()
	if t == nil || !t.IsFuncArgStruct() {
		return
	}

	// Rewrite f(g()) into t1, t2, ... = g(); f(t1, t2, ...).

	// Save n as n.Orig for fmt.go.
	if ir.Orig(n) == n {
		n.(ir.OrigNode).SetOrig(ir.SepCopy(n))
	}

	as := ir.NewAssignListStmt(base.Pos, ir.OAS2, nil, nil)
	as.Rhs.Append(list...)

	// If we're outside of function context, then this call will
	// be executed during the generated init function. However,
	// init.go hasn't yet created it. Instead, associate the
	// temporary variables with  InitTodoFunc for now, and init.go
	// will reassociate them later when it's appropriate.
	static := ir.CurFunc == nil
	if static {
		ir.CurFunc = typecheck.InitTodoFunc
	}
	list = nil
	for _, f := range t.FieldSlice() {
		t := typecheck.Temp(f.Type)
		as.PtrInit().Append(ir.NewDecl(base.Pos, ir.ODCL, t))
		as.Lhs.Append(t)
		list = append(list, t)
	}
	if static {
		ir.CurFunc = nil
	}

	switch n := n.(type) {
	case *ir.CallExpr:
		n.Args = list
	case *ir.ReturnStmt:
		n.Results = list
	}

	transformAssign(as, as.Lhs, as.Rhs)
	as.SetTypecheck(1)
	n.PtrInit().Append(as)
}

// assignconvfn converts node n for assignment to type t. Corresponds to
// typecheck.assignconvfn.
func assignconvfn(n ir.Node, t *types.Type) ir.Node {
	if t.Kind() == types.TBLANK {
		return n
	}

	if types.Identical(n.Type(), t) {
		return n
	}

	op, _ := typecheck.Assignop(n.Type(), t)

	r := ir.NewConvExpr(base.Pos, op, t, n)
	r.SetTypecheck(1)
	r.SetImplicit(true)
	return r
}

// Corresponds to typecheck.typecheckaste.
func typecheckaste(op ir.Op, call ir.Node, isddd bool, tstruct *types.Type, nl ir.Nodes) {
	var t *types.Type
	var i int

	lno := base.Pos
	defer func() { base.Pos = lno }()

	var n ir.Node
	if len(nl) == 1 {
		n = nl[0]
	}

	i = 0
	for _, tl := range tstruct.Fields().Slice() {
		t = tl.Type
		if tl.IsDDD() {
			if isddd {
				n = nl[i]
				ir.SetPos(n)
				if n.Type() != nil {
					nl[i] = assignconvfn(n, t)
				}
				return
			}

			// TODO(mdempsky): Make into ... call with implicit slice.
			for ; i < len(nl); i++ {
				n = nl[i]
				ir.SetPos(n)
				if n.Type() != nil {
					nl[i] = assignconvfn(n, t.Elem())
				}
			}
			return
		}

		n = nl[i]
		ir.SetPos(n)
		if n.Type() != nil {
			nl[i] = assignconvfn(n, t)
		}
		i++
	}
}

// transformSend transforms a send statement, converting the value to appropriate
// type for the channel, as needed. Corresponds of typecheck.tcSend.
func transformSend(n *ir.SendStmt) {
	n.Value = assignconvfn(n.Value, n.Chan.Type().Elem())
}

// transformReturn transforms a return node, by doing the needed assignments and
// any necessary conversions. Corresponds to typecheck.tcReturn()
func transformReturn(rs *ir.ReturnStmt) {
	transformArgs(rs)
	nl := rs.Results
	if ir.HasNamedResults(ir.CurFunc) && len(nl) == 0 {
		return
	}

	typecheckaste(ir.ORETURN, nil, false, ir.CurFunc.Type().Results(), nl)
}

// transformSelect transforms a select node, creating an assignment list as needed
// for each case. Corresponds to typecheck.tcSelect().
func transformSelect(sel *ir.SelectStmt) {
	for _, ncase := range sel.Cases {
		if ncase.Comm != nil {
			n := ncase.Comm
			oselrecv2 := func(dst, recv ir.Node, def bool) {
				n := ir.NewAssignListStmt(n.Pos(), ir.OSELRECV2, []ir.Node{dst, ir.BlankNode}, []ir.Node{recv})
				n.Def = def
				n.SetTypecheck(1)
				ncase.Comm = n
			}
			switch n.Op() {
			case ir.OAS:
				// convert x = <-c into x, _ = <-c
				// remove implicit conversions; the eventual assignment
				// will reintroduce them.
				n := n.(*ir.AssignStmt)
				if r := n.Y; r.Op() == ir.OCONVNOP || r.Op() == ir.OCONVIFACE {
					r := r.(*ir.ConvExpr)
					if r.Implicit() {
						n.Y = r.X
					}
				}
				oselrecv2(n.X, n.Y, n.Def)

			case ir.OAS2RECV:
				n := n.(*ir.AssignListStmt)
				n.SetOp(ir.OSELRECV2)

			case ir.ORECV:
				// convert <-c into _, _ = <-c
				n := n.(*ir.UnaryExpr)
				oselrecv2(ir.BlankNode, n, false)

			case ir.OSEND:
				break
			}
		}
	}
}

// transformAsOp transforms an AssignOp statement. Corresponds to OASOP case in
// typecheck1.
func transformAsOp(n *ir.AssignOpStmt) {
	transformCheckAssign(n, n.X)
}

// transformDot transforms an OXDOT (or ODOT) or ODOT, ODOTPTR, ODOTMETH,
// ODOTINTER, or OCALLPART, as appropriate. It adds in extra nodes as needed to
// access embedded fields. Corresponds to typecheck.tcDot.
func transformDot(n *ir.SelectorExpr, isCall bool) ir.Node {
	assert(n.Type() != nil && n.Typecheck() == 1)
	if n.Op() == ir.OXDOT {
		n = typecheck.AddImplicitDots(n)
		n.SetOp(ir.ODOT)
	}

	t := n.X.Type()

	if n.X.Op() == ir.OTYPE {
		return transformMethodExpr(n)
	}

	if t.IsPtr() && !t.Elem().IsInterface() {
		t = t.Elem()
		n.SetOp(ir.ODOTPTR)
	}

	f := typecheck.Lookdot(n, t, 0)
	assert(f != nil)

	if (n.Op() == ir.ODOTINTER || n.Op() == ir.ODOTMETH) && !isCall {
		n.SetOp(ir.OCALLPART)
		n.SetType(typecheck.MethodValueWrapper(n).Type())
	}
	return n
}

// Corresponds to typecheck.typecheckMethodExpr.
func transformMethodExpr(n *ir.SelectorExpr) (res ir.Node) {
	t := n.X.Type()

	// Compute the method set for t.
	var ms *types.Fields
	if t.IsInterface() {
		ms = t.AllMethods()
	} else {
		mt := types.ReceiverBaseType(t)
		typecheck.CalcMethods(mt)
		ms = mt.AllMethods()

		// The method expression T.m requires a wrapper when T
		// is different from m's declared receiver type. We
		// normally generate these wrappers while writing out
		// runtime type descriptors, which is always done for
		// types declared at package scope. However, we need
		// to make sure to generate wrappers for anonymous
		// receiver types too.
		if mt.Sym() == nil {
			typecheck.NeedRuntimeType(t)
		}
	}

	s := n.Sel
	m := typecheck.Lookdot1(n, s, t, ms, 0)
	assert(m != nil)

	n.SetOp(ir.OMETHEXPR)
	n.Selection = m
	n.SetType(typecheck.NewMethodType(m.Type, n.X.Type()))
	return n
}

// Corresponds to typecheck.tcAppend.
func transformAppend(n *ir.CallExpr) ir.Node {
	transformArgs(n)
	args := n.Args
	t := args[0].Type()
	assert(t.IsSlice())

	if n.IsDDD {
		if t.Elem().IsKind(types.TUINT8) && args[1].Type().IsString() {
			return n
		}

		args[1] = assignconvfn(args[1], t.Underlying())
		return n
	}

	as := args[1:]
	for i, n := range as {
		assert(n.Type() != nil)
		as[i] = assignconvfn(n, t.Elem())
	}
	return n
}

// Corresponds to typecheck.tcComplex.
func transformComplex(n *ir.BinaryExpr) ir.Node {
	l := n.X
	r := n.Y

	assert(types.Identical(l.Type(), r.Type()))

	var t *types.Type
	switch l.Type().Kind() {
	case types.TFLOAT32:
		t = types.Types[types.TCOMPLEX64]
	case types.TFLOAT64:
		t = types.Types[types.TCOMPLEX128]
	default:
		panic(fmt.Sprintf("transformComplex: unexpected type %v", l.Type()))
	}

	// Must set the type here for generics, because this can't be determined
	// by substitution of the generic types.
	typed(t, n)
	return n
}

// Corresponds to typecheck.tcDelete.
func transformDelete(n *ir.CallExpr) ir.Node {
	transformArgs(n)
	args := n.Args
	assert(len(args) == 2)

	l := args[0]
	r := args[1]

	args[1] = assignconvfn(r, l.Type().Key())
	return n
}

// Corresponds to typecheck.tcMake.
func transformMake(n *ir.CallExpr) ir.Node {
	args := n.Args

	n.Args = nil
	l := args[0]
	t := l.Type()
	assert(t != nil)

	i := 1
	var nn ir.Node
	switch t.Kind() {
	case types.TSLICE:
		l = args[i]
		i++
		var r ir.Node
		if i < len(args) {
			r = args[i]
			i++
		}
		nn = ir.NewMakeExpr(n.Pos(), ir.OMAKESLICE, l, r)

	case types.TMAP:
		if i < len(args) {
			l = args[i]
			i++
		} else {
			l = ir.NewInt(0)
		}
		nn = ir.NewMakeExpr(n.Pos(), ir.OMAKEMAP, l, nil)
		nn.SetEsc(n.Esc())

	case types.TCHAN:
		l = nil
		if i < len(args) {
			l = args[i]
			i++
		} else {
			l = ir.NewInt(0)
		}
		nn = ir.NewMakeExpr(n.Pos(), ir.OMAKECHAN, l, nil)
	default:
		panic(fmt.Sprintf("transformMake: unexpected type %v", t))
	}

	assert(i == len(args))
	typed(n.Type(), nn)
	return nn
}

// Corresponds to typecheck.tcPanic.
func transformPanic(n *ir.UnaryExpr) ir.Node {
	n.X = assignconvfn(n.X, types.Types[types.TINTER])
	return n
}

// Corresponds to typecheck.tcPrint.
func transformPrint(n *ir.CallExpr) ir.Node {
	transformArgs(n)
	return n
}

// Corresponds to typecheck.tcRealImag.
func transformRealImag(n *ir.UnaryExpr) ir.Node {
	l := n.X
	var t *types.Type

	// Determine result type.
	switch l.Type().Kind() {
	case types.TCOMPLEX64:
		t = types.Types[types.TFLOAT32]
	case types.TCOMPLEX128:
		t = types.Types[types.TFLOAT64]
	default:
		panic(fmt.Sprintf("transformRealImag: unexpected type %v", l.Type()))
	}

	// Must set the type here for generics, because this can't be determined
	// by substitution of the generic types.
	typed(t, n)
	return n
}

// Corresponds to typecheck.tcLenCap.
func transformLenCap(n *ir.UnaryExpr) ir.Node {
	n.X = implicitstar(n.X)
	return n
}

// Corresponds to Builtin part of tcCall.
func transformBuiltin(n *ir.CallExpr) ir.Node {
	// n.Type() can be nil for builtins with no return value
	assert(n.Typecheck() == 1)
	fun := n.X.(*ir.Name)
	op := fun.BuiltinOp

	switch op {
	case ir.OAPPEND, ir.ODELETE, ir.OMAKE, ir.OPRINT, ir.OPRINTN, ir.ORECOVER:
		n.SetOp(op)
		n.X = nil
		switch op {
		case ir.OAPPEND:
			return transformAppend(n)
		case ir.ODELETE:
			return transformDelete(n)
		case ir.OMAKE:
			return transformMake(n)
		case ir.OPRINT, ir.OPRINTN:
			return transformPrint(n)
		case ir.ORECOVER:
			// nothing more to do
			return n
		}

	case ir.OCAP, ir.OCLOSE, ir.OIMAG, ir.OLEN, ir.OPANIC, ir.OREAL:
		transformArgs(n)
		fallthrough

	case ir.ONEW, ir.OALIGNOF, ir.OOFFSETOF, ir.OSIZEOF:
		u := ir.NewUnaryExpr(n.Pos(), op, n.Args[0])
		u1 := typed(n.Type(), ir.InitExpr(n.Init(), u)) // typecheckargs can add to old.Init
		switch op {
		case ir.OCAP, ir.OLEN:
			return transformLenCap(u1.(*ir.UnaryExpr))
		case ir.OREAL, ir.OIMAG:
			return transformRealImag(u1.(*ir.UnaryExpr))
		case ir.OPANIC:
			return transformPanic(u1.(*ir.UnaryExpr))
		case ir.OCLOSE, ir.ONEW, ir.OALIGNOF, ir.OOFFSETOF, ir.OSIZEOF:
			// nothing more to do
			return u1
		}

	case ir.OCOMPLEX, ir.OCOPY, ir.OUNSAFEADD, ir.OUNSAFESLICE:
		transformArgs(n)
		b := ir.NewBinaryExpr(n.Pos(), op, n.Args[0], n.Args[1])
		n1 := typed(n.Type(), ir.InitExpr(n.Init(), b))
		if op != ir.OCOMPLEX {
			// nothing more to do
			return n1
		}
		return transformComplex(n1.(*ir.BinaryExpr))

	default:
		panic(fmt.Sprintf("transformBuiltin: unexpected op %v", op))
	}

	return n
}

func hasKeys(l ir.Nodes) bool {
	for _, n := range l {
		if n.Op() == ir.OKEY || n.Op() == ir.OSTRUCTKEY {
			return true
		}
	}
	return false
}

// transformArrayLit runs assignconvfn on each array element and returns the
// length of the slice/array that is needed to hold all the array keys/indexes
// (one more than the highest index). Corresponds to typecheck.typecheckarraylit.
func transformArrayLit(elemType *types.Type, bound int64, elts []ir.Node) int64 {
	var key, length int64
	for i, elt := range elts {
		ir.SetPos(elt)
		r := elts[i]
		var kv *ir.KeyExpr
		if elt.Op() == ir.OKEY {
			elt := elt.(*ir.KeyExpr)
			key = typecheck.IndexConst(elt.Key)
			assert(key >= 0)
			kv = elt
			r = elt.Value
		}

		r = assignconvfn(r, elemType)
		if kv != nil {
			kv.Value = r
		} else {
			elts[i] = r
		}

		key++
		if key > length {
			length = key
		}
	}

	return length
}

// transformCompLit transforms n to an OARRAYLIT, OSLICELIT, OMAPLIT, or
// OSTRUCTLIT node, with any needed conversions. Corresponds to
// typecheck.tcCompLit.
func transformCompLit(n *ir.CompLitExpr) (res ir.Node) {
	assert(n.Type() != nil && n.Typecheck() == 1)
	lno := base.Pos
	defer func() {
		base.Pos = lno
	}()

	// Save original node (including n.Right)
	n.SetOrig(ir.Copy(n))

	ir.SetPos(n)

	t := n.Type()

	switch t.Kind() {
	default:
		base.Fatalf("transformCompLit %v", t.Kind())

	case types.TARRAY:
		transformArrayLit(t.Elem(), t.NumElem(), n.List)
		n.SetOp(ir.OARRAYLIT)

	case types.TSLICE:
		length := transformArrayLit(t.Elem(), -1, n.List)
		n.SetOp(ir.OSLICELIT)
		n.Len = length

	case types.TMAP:
		for _, l := range n.List {
			ir.SetPos(l)
			assert(l.Op() == ir.OKEY)
			l := l.(*ir.KeyExpr)

			r := l.Key
			l.Key = assignconvfn(r, t.Key())

			r = l.Value
			l.Value = assignconvfn(r, t.Elem())
		}

		n.SetOp(ir.OMAPLIT)

	case types.TSTRUCT:
		// Need valid field offsets for Xoffset below.
		types.CalcSize(t)

		if len(n.List) != 0 && !hasKeys(n.List) {
			// simple list of values
			ls := n.List
			for i, n1 := range ls {
				ir.SetPos(n1)

				f := t.Field(i)
				n1 = assignconvfn(n1, f.Type)
				sk := ir.NewStructKeyExpr(base.Pos, f.Sym, n1)
				sk.Offset = f.Offset
				ls[i] = sk
			}
			assert(len(ls) >= t.NumFields())
		} else {
			// keyed list
			ls := n.List
			for i, l := range ls {
				ir.SetPos(l)

				if l.Op() == ir.OKEY {
					kv := l.(*ir.KeyExpr)
					key := kv.Key

					// Sym might have resolved to name in other top-level
					// package, because of import dot. Redirect to correct sym
					// before we do the lookup.
					s := key.Sym()
					if id, ok := key.(*ir.Ident); ok && typecheck.DotImportRefs[id] != nil {
						s = typecheck.Lookup(s.Name)
					}

					// An OXDOT uses the Sym field to hold
					// the field to the right of the dot,
					// so s will be non-nil, but an OXDOT
					// is never a valid struct literal key.
					assert(!(s == nil || s.Pkg != types.LocalPkg || key.Op() == ir.OXDOT || s.IsBlank()))

					l = ir.NewStructKeyExpr(l.Pos(), s, kv.Value)
					ls[i] = l
				}

				assert(l.Op() == ir.OSTRUCTKEY)
				l := l.(*ir.StructKeyExpr)

				f := typecheck.Lookdot1(nil, l.Field, t, t.Fields(), 0)
				l.Offset = f.Offset

				l.Value = assignconvfn(l.Value, f.Type)
			}
		}

		n.SetOp(ir.OSTRUCTLIT)
	}

	return n
}
