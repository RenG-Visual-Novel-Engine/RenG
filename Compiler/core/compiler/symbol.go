package compiler

import (
	"RenG/Compiler/core/code"
	"RenG/Compiler/core/object"
	"fmt"
)

func (c *Compiler) loadSymbol(s Symbol) {
	switch s.Scope {
	case GlobalScope:
		c.emit(code.OpGetGlobal, s.Index)
	case LocalScope:
		c.emit(code.OpGetLocal, s.Index)
	case BuiltinScope:
		c.emit(code.OpGetBuiltin, s.Index)
	}
}

func (c *Compiler) ReplaceSymbol() error {
	for _, inform := range c.reservationSymbol {

		fn := c.constants[inform.ReplaceFuncIndex]
		fnObj, ok := fn.(*object.CompiledFunction)
		if !ok {
			return fmt.Errorf("")
		}

		s, ok := c.symbolTable.Resolve(inform.symbol)
		if !ok {
			return fmt.Errorf("")
		}

		op := code.Make(code.OpGetGlobal, s.Index)

		for n := 0; n < len(op); n++ {
			fnObj.Instructions[inform.pos+n] = op[n]
		}
		c.constants[inform.ReplaceFuncIndex] = fnObj
	}

	return nil
}
