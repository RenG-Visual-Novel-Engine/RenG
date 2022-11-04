package compiler

import (
	"RenG/Compiler/core/ast"
	"RenG/Compiler/core/code"
)

func (c *Compiler) compExpressionStatement(es *ast.ExpressionStatement) error {
	err := c.Compile(es.Expression)
	if err != nil {
		return err
	}

	if !c.lastInsructionIs(code.OpSetGlobal) {
		c.emit(code.OpPop)
	}

	return nil
}
