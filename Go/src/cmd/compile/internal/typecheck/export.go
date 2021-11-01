// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package typecheck

import (
	"go/constant"

	"cmd/compile/internal/base"
	"cmd/compile/internal/ir"
	"cmd/compile/internal/types"
	"cmd/internal/src"
)

// importalias declares symbol s as an imported type alias with type t.
// ipkg is the package being imported
func importalias(ipkg *types.Pkg, pos src.XPos, s *types.Sym, t *types.Type) *ir.Name {
	return importobj(ipkg, pos, s, ir.OTYPE, ir.PEXTERN, t)
}

// importconst declares symbol s as an imported constant with type t and value val.
// ipkg is the package being imported
func importconst(ipkg *types.Pkg, pos src.XPos, s *types.Sym, t *types.Type, val constant.Value) *ir.Name {
	n := importobj(ipkg, pos, s, ir.OLITERAL, ir.PEXTERN, t)
	n.SetVal(val)
	return n
}

// importfunc declares symbol s as an imported function with type t.
// ipkg is the package being imported
func importfunc(ipkg *types.Pkg, pos src.XPos, s *types.Sym, t *types.Type) *ir.Name {
	n := importobj(ipkg, pos, s, ir.ONAME, ir.PFUNC, t)
	n.Func = ir.NewFunc(pos)
	n.Func.Nname = n
	return n
}

// importobj declares symbol s as an imported object representable by op.
// ipkg is the package being imported
func importobj(ipkg *types.Pkg, pos src.XPos, s *types.Sym, op ir.Op, ctxt ir.Class, t *types.Type) *ir.Name {
	n := importsym(ipkg, pos, s, op, ctxt)
	n.SetType(t)
	if ctxt == ir.PFUNC {
		n.Sym().SetFunc(true)
	}
	return n
}

func importsym(ipkg *types.Pkg, pos src.XPos, s *types.Sym, op ir.Op, ctxt ir.Class) *ir.Name {
	if n := s.PkgDef(); n != nil {
		base.Fatalf("importsym of symbol that already exists: %v", n)
	}

	n := ir.NewDeclNameAt(pos, op, s)
	n.Class = ctxt // TODO(mdempsky): Move this into NewDeclNameAt too?
	s.SetPkgDef(n)
	return n
}

// importtype returns the named type declared by symbol s.
// If no such type has been declared yet, a forward declaration is returned.
// ipkg is the package being imported
func importtype(ipkg *types.Pkg, pos src.XPos, s *types.Sym) *ir.Name {
	n := importsym(ipkg, pos, s, ir.OTYPE, ir.PEXTERN)
	n.SetType(types.NewNamed(n))
	return n
}

// importvar declares symbol s as an imported variable with type t.
// ipkg is the package being imported
func importvar(ipkg *types.Pkg, pos src.XPos, s *types.Sym, t *types.Type) *ir.Name {
	return importobj(ipkg, pos, s, ir.ONAME, ir.PEXTERN, t)
}
