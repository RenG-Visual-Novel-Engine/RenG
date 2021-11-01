// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package noder

import (
	"cmd/compile/internal/ir"
	"cmd/compile/internal/syntax"
	"cmd/compile/internal/typecheck"
	"cmd/compile/internal/types"
	"cmd/internal/src"
)

func (g *irgen) stmts(stmts []syntax.Stmt) []ir.Node {
	var nodes []ir.Node
	for _, stmt := range stmts {
		switch s := g.stmt(stmt).(type) {
		case nil: // EmptyStmt
		case *ir.BlockStmt:
			nodes = append(nodes, s.List...)
		default:
			nodes = append(nodes, s)
		}
	}
	return nodes
}

func (g *irgen) stmt(stmt syntax.Stmt) ir.Node {
	switch stmt := stmt.(type) {
	case nil, *syntax.EmptyStmt:
		return nil
	case *syntax.LabeledStmt:
		return g.labeledStmt(stmt)
	case *syntax.BlockStmt:
		return ir.NewBlockStmt(g.pos(stmt), g.blockStmt(stmt))
	case *syntax.ExprStmt:
		x := g.expr(stmt.X)
		if call, ok := x.(*ir.CallExpr); ok {
			call.Use = ir.CallUseStmt
		}
		return x
	case *syntax.SendStmt:
		n := ir.NewSendStmt(g.pos(stmt), g.expr(stmt.Chan), g.expr(stmt.Value))
		if n.Chan.Type().HasTParam() || n.Value.Type().HasTParam() {
			// Delay transforming the send if the channel or value
			// have a type param.
			n.SetTypecheck(3)
			return n
		}
		transformSend(n)
		n.SetTypecheck(1)
		return n
	case *syntax.DeclStmt:
		return ir.NewBlockStmt(g.pos(stmt), g.decls(stmt.DeclList))

	case *syntax.AssignStmt:
		if stmt.Op != 0 && stmt.Op != syntax.Def {
			op := g.op(stmt.Op, binOps[:])
			var n *ir.AssignOpStmt
			if stmt.Rhs == nil {
				n = IncDec(g.pos(stmt), op, g.expr(stmt.Lhs))
			} else {
				n = ir.NewAssignOpStmt(g.pos(stmt), op, g.expr(stmt.Lhs), g.expr(stmt.Rhs))
			}
			if n.X.Typecheck() == 3 {
				n.SetTypecheck(3)
				return n
			}
			transformAsOp(n)
			n.SetTypecheck(1)
			return n
		}

		names, lhs := g.assignList(stmt.Lhs, stmt.Op == syntax.Def)
		rhs := g.exprList(stmt.Rhs)

		// We must delay transforming the assign statement if any of the
		// lhs or rhs nodes are also delayed, since transformAssign needs
		// to know the types of the left and right sides in various cases.
		delay := false
		for _, e := range lhs {
			if e.Typecheck() == 3 {
				delay = true
				break
			}
		}
		for _, e := range rhs {
			if e.Typecheck() == 3 {
				delay = true
				break
			}
		}

		if len(lhs) == 1 && len(rhs) == 1 {
			n := ir.NewAssignStmt(g.pos(stmt), lhs[0], rhs[0])
			n.Def = initDefn(n, names)

			if delay {
				n.SetTypecheck(3)
				return n
			}

			lhs, rhs := []ir.Node{n.X}, []ir.Node{n.Y}
			transformAssign(n, lhs, rhs)
			n.X, n.Y = lhs[0], rhs[0]
			n.SetTypecheck(1)
			return n
		}

		n := ir.NewAssignListStmt(g.pos(stmt), ir.OAS2, lhs, rhs)
		n.Def = initDefn(n, names)
		if delay {
			n.SetTypecheck(3)
			return n
		}
		transformAssign(n, n.Lhs, n.Rhs)
		n.SetTypecheck(1)
		return n

	case *syntax.BranchStmt:
		return ir.NewBranchStmt(g.pos(stmt), g.tokOp(int(stmt.Tok), branchOps[:]), g.name(stmt.Label))
	case *syntax.CallStmt:
		return ir.NewGoDeferStmt(g.pos(stmt), g.tokOp(int(stmt.Tok), callOps[:]), g.expr(stmt.Call))
	case *syntax.ReturnStmt:
		n := ir.NewReturnStmt(g.pos(stmt), g.exprList(stmt.Results))
		for _, e := range n.Results {
			if e.Type().HasTParam() {
				// Delay transforming the return statement if any of the
				// return values have a type param.
				n.SetTypecheck(3)
				return n
			}
		}
		transformReturn(n)
		n.SetTypecheck(1)
		return n
	case *syntax.IfStmt:
		return g.ifStmt(stmt)
	case *syntax.ForStmt:
		return g.forStmt(stmt)
	case *syntax.SelectStmt:
		n := g.selectStmt(stmt)
		transformSelect(n.(*ir.SelectStmt))
		n.SetTypecheck(1)
		return n
	case *syntax.SwitchStmt:
		return g.switchStmt(stmt)

	default:
		g.unhandled("statement", stmt)
		panic("unreachable")
	}
}

// TODO(mdempsky): Investigate replacing with switch statements or dense arrays.

var branchOps = [...]ir.Op{
	syntax.Break:       ir.OBREAK,
	syntax.Continue:    ir.OCONTINUE,
	syntax.Fallthrough: ir.OFALL,
	syntax.Goto:        ir.OGOTO,
}

var callOps = [...]ir.Op{
	syntax.Defer: ir.ODEFER,
	syntax.Go:    ir.OGO,
}

func (g *irgen) tokOp(tok int, ops []ir.Op) ir.Op {
	// TODO(mdempsky): Validate.
	return ops[tok]
}

func (g *irgen) op(op syntax.Operator, ops []ir.Op) ir.Op {
	// TODO(mdempsky): Validate.
	return ops[op]
}

func (g *irgen) assignList(expr syntax.Expr, def bool) ([]*ir.Name, []ir.Node) {
	if !def {
		return nil, g.exprList(expr)
	}

	var exprs []syntax.Expr
	if list, ok := expr.(*syntax.ListExpr); ok {
		exprs = list.ElemList
	} else {
		exprs = []syntax.Expr{expr}
	}

	var names []*ir.Name
	res := make([]ir.Node, len(exprs))
	for i, expr := range exprs {
		expr := expr.(*syntax.Name)
		if expr.Value == "_" {
			res[i] = ir.BlankNode
			continue
		}

		if obj, ok := g.info.Uses[expr]; ok {
			res[i] = g.obj(obj)
			continue
		}

		name, _ := g.def(expr)
		names = append(names, name)
		res[i] = name
	}

	return names, res
}

// initDefn marks the given names as declared by defn and populates
// its Init field with ODCL nodes. It then reports whether any names
// were so declared, which can be used to initialize defn.Def.
func initDefn(defn ir.InitNode, names []*ir.Name) bool {
	if len(names) == 0 {
		return false
	}

	init := make([]ir.Node, len(names))
	for i, name := range names {
		name.Defn = defn
		init[i] = ir.NewDecl(name.Pos(), ir.ODCL, name)
	}
	defn.SetInit(init)
	return true
}

func (g *irgen) blockStmt(stmt *syntax.BlockStmt) []ir.Node {
	return g.stmts(stmt.List)
}

func (g *irgen) ifStmt(stmt *syntax.IfStmt) ir.Node {
	init := g.stmt(stmt.Init)
	n := ir.NewIfStmt(g.pos(stmt), g.expr(stmt.Cond), g.blockStmt(stmt.Then), nil)
	if stmt.Else != nil {
		e := g.stmt(stmt.Else)
		if e.Op() == ir.OBLOCK {
			e := e.(*ir.BlockStmt)
			n.Else = e.List
		} else {
			n.Else = []ir.Node{e}
		}
	}
	return g.init(init, n)
}

// unpackTwo returns the first two nodes in list. If list has fewer
// than 2 nodes, then the missing nodes are replaced with nils.
func unpackTwo(list []ir.Node) (fst, snd ir.Node) {
	switch len(list) {
	case 0:
		return nil, nil
	case 1:
		return list[0], nil
	default:
		return list[0], list[1]
	}
}

func (g *irgen) forStmt(stmt *syntax.ForStmt) ir.Node {
	if r, ok := stmt.Init.(*syntax.RangeClause); ok {
		names, lhs := g.assignList(r.Lhs, r.Def)
		key, value := unpackTwo(lhs)
		n := ir.NewRangeStmt(g.pos(r), key, value, g.expr(r.X), g.blockStmt(stmt.Body))
		n.Def = initDefn(n, names)
		return n
	}

	return ir.NewForStmt(g.pos(stmt), g.stmt(stmt.Init), g.expr(stmt.Cond), g.stmt(stmt.Post), g.blockStmt(stmt.Body))
}

func (g *irgen) selectStmt(stmt *syntax.SelectStmt) ir.Node {
	body := make([]*ir.CommClause, len(stmt.Body))
	for i, clause := range stmt.Body {
		body[i] = ir.NewCommStmt(g.pos(clause), g.stmt(clause.Comm), g.stmts(clause.Body))
	}
	return ir.NewSelectStmt(g.pos(stmt), body)
}

func (g *irgen) switchStmt(stmt *syntax.SwitchStmt) ir.Node {
	pos := g.pos(stmt)
	init := g.stmt(stmt.Init)

	var expr ir.Node
	switch tag := stmt.Tag.(type) {
	case *syntax.TypeSwitchGuard:
		var ident *ir.Ident
		if tag.Lhs != nil {
			ident = ir.NewIdent(g.pos(tag.Lhs), g.name(tag.Lhs))
		}
		expr = ir.NewTypeSwitchGuard(pos, ident, g.expr(tag.X))
	default:
		expr = g.expr(tag)
	}

	body := make([]*ir.CaseClause, len(stmt.Body))
	for i, clause := range stmt.Body {
		// Check for an implicit clause variable before
		// visiting body, because it may contain function
		// literals that reference it, and then it'll be
		// associated to the wrong function.
		//
		// Also, override its position to the clause's colon, so that
		// dwarfgen can find the right scope for it later.
		// TODO(mdempsky): We should probably just store the scope
		// directly in the ir.Name.
		var cv *ir.Name
		if obj, ok := g.info.Implicits[clause]; ok {
			cv = g.obj(obj)
			cv.SetPos(g.makeXPos(clause.Colon))
		}
		body[i] = ir.NewCaseStmt(g.pos(clause), g.exprList(clause.Cases), g.stmts(clause.Body))
		body[i].Var = cv
	}

	return g.init(init, ir.NewSwitchStmt(pos, expr, body))
}

func (g *irgen) labeledStmt(label *syntax.LabeledStmt) ir.Node {
	sym := g.name(label.Label)
	lhs := ir.NewLabelStmt(g.pos(label), sym)
	ls := g.stmt(label.Stmt)

	// Attach label directly to control statement too.
	switch ls := ls.(type) {
	case *ir.ForStmt:
		ls.Label = sym
	case *ir.RangeStmt:
		ls.Label = sym
	case *ir.SelectStmt:
		ls.Label = sym
	case *ir.SwitchStmt:
		ls.Label = sym
	}

	l := []ir.Node{lhs}
	if ls != nil {
		if ls.Op() == ir.OBLOCK {
			ls := ls.(*ir.BlockStmt)
			l = append(l, ls.List...)
		} else {
			l = append(l, ls)
		}
	}
	return ir.NewBlockStmt(src.NoXPos, l)
}

func (g *irgen) init(init ir.Node, stmt ir.InitNode) ir.InitNode {
	if init != nil {
		stmt.SetInit([]ir.Node{init})
	}
	return stmt
}

func (g *irgen) name(name *syntax.Name) *types.Sym {
	if name == nil {
		return nil
	}
	return typecheck.Lookup(name.Value)
}
