package compiler

import (
	"RenG/Compiler/core/ast"
	"RenG/Compiler/core/code"
	"RenG/Compiler/core/object"
	"fmt"
)

func (c *Compiler) compObjectProgram(p *ast.Program) error {
	for _, s := range p.Statements {
		err := c.CompileObject(s)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Compiler) compObjectExpressionStatement(es *ast.ExpressionStatement) error {
	err := c.CompileObject(es.Expression)
	if err != nil {
		return err
	}

	if !c.lastInsructionIs(code.OpSetGlobal) && !c.lastInsructionIs(code.OpSetLocal) {
		c.emit(code.OpPop)
	}

	return nil
}

func (c *Compiler) compObjectBlockStatement(bs *ast.BlockStatement) error {
	for _, s := range bs.Statements {
		err := c.CompileObject(s)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Compiler) compObjectFunctionStatement(fs *ast.FunctionStatement) error {
	symbol := c.symbolTable.Define(fs.Name.Value)

	c.enterScope()

	for _, p := range fs.Parameters {
		c.symbolTable.Define(p.Value)
	}

	err := c.CompileObject(fs.Body)
	if err != nil {
		return err
	}

	if c.lastInsructionIs(code.OpPop) {
		c.replaceLastPopWithReturn()
	}

	if !c.lastInsructionIs(code.OpReturnValue) {
		c.emit(code.OpReturn)
	}

	numLocals := c.symbolTable.numDefinitions
	instructions := c.leaveScope()

	compiledFn := &object.CompiledFunction{
		Instructions:  instructions,
		NumLocals:     numLocals,
		NumParameters: len(fs.Parameters),
	}

	c.emit(code.OpConstant, c.addConstant(compiledFn))
	c.emit(code.OpSetGlobal, symbol.Index)

	return nil
}

func (c *Compiler) compObjectIfStatement(is *ast.IfStatement) error {
	var jmpPos []int

	err := c.CompileObject(is.Condition)
	if err != nil {
		return err
	}

	jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

	err = c.CompileObject(is.Consequence)
	if err != nil {
		return err
	}

	if c.lastInsructionIs(code.OpPop) {
		c.removeLastPop()
	}

	if len(is.Elif) == 0 && is.Else == nil {
		afterConsequencePos := len(c.currentInstructions())
		c.changeOperand(jumpNotTruthyPos, afterConsequencePos)
	} else if len(is.Elif) != 0 {

		for _, elif := range is.Elif {
			jmpPos = append(jmpPos, c.emit(code.OpJump, 9999))

			afterConsequencePos := len(c.currentInstructions())
			c.changeOperand(jumpNotTruthyPos, afterConsequencePos)

			err := c.CompileObject(elif.Condition)
			if err != nil {
				return err
			}

			jumpNotTruthyPos = c.emit(code.OpJumpNotTruthy, 9999)

			err = c.CompileObject(elif.Consequence)
			if err != nil {
				return err
			}

			if c.lastInsructionIs(code.OpPop) {
				c.removeLastPop()
			}
		}
	}

	jmpPos = append(jmpPos, c.emit(code.OpJump, 9999))

	afterConsequencePos := len(c.currentInstructions())
	c.changeOperand(jumpNotTruthyPos, afterConsequencePos)

	if is.Else != nil {
		err := c.CompileObject(is.Else)
		if err != nil {
			return err
		}

		if c.lastInsructionIs(code.OpPop) {
			c.removeLastPop()
		}
	} else {
		c.emit(code.OpNull)
		c.emit(code.OpPop)
	}

	afterElsePos := len(c.currentInstructions())
	for _, jmp := range jmpPos {
		c.changeOperand(jmp, afterElsePos)
	}

	return nil
}

func (c *Compiler) compObjectForStatement(fs *ast.ForStatement) error {
	var jmpNotTruthyPos, jmpPos int = -1, -1

	if fs.Initialization != nil {
		err := c.CompileObject(fs.Initialization)
		if err != nil {
			return err
		}
	}

	jmpPos = len(c.currentInstructions())

	if fs.Condition != nil {
		err := c.CompileObject(fs.Condition)
		if err != nil {
			return err
		}

		jmpNotTruthyPos = c.emit(code.OpJumpNotTruthy, 9999)
	}

	err := c.CompileObject(fs.Loop)
	if err != nil {
		return err
	}

	if fs.Increment != nil {
		err := c.CompileObject(fs.Increment)
		if err != nil {
			return err
		}
	}

	c.emit(code.OpJump, jmpPos)

	if jmpNotTruthyPos != -1 {
		c.changeOperand(jmpNotTruthyPos, len(c.currentInstructions()))
	}

	return nil
}

func (c *Compiler) compObjectReturnStatement(rs *ast.ReturnStatement) error {
	err := c.CompileObject(rs.ReturnValue)
	if err != nil {
		return err
	}

	c.emit(code.OpReturnValue)

	return nil
}

func (c *Compiler) compObjectPrefixExpression(pe *ast.PrefixExpression) error {
	err := c.CompileObject(pe.Right)
	if err != nil {
		return err
	}

	switch pe.Operator {
	case "-":
		c.emit(code.OpMinus)
	case "!":
		c.emit(code.OpBang)
	default:
		return fmt.Errorf("알 수 없는 전위연산자 [%s]", pe.Operator)
	}

	return nil
}

func (c *Compiler) compObjectInfixExpression(ie *ast.InfixExpression) error {

	switch ie.Operator {
	case "=":
		symbol, ok := c.symbolTable.Resolve(ie.Left.TokenLiteral())
		if !ok {
			symbol = c.symbolTable.Define(ie.Left.TokenLiteral())
		}

		err := c.CompileObject(ie.Right)
		if err != nil {
			return err
		}

		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
		}

		return nil
	case "+=":
		ident, ok := ie.Left.(*ast.Identifier)
		if !ok {
			return fmt.Errorf("+= 연산자 왼쪽 값이 변수가 아닙니다. %v", ident)
		}

		symbol, ok := c.symbolTable.Resolve(ident.Value)
		if !ok {
			return fmt.Errorf("정의되지 않은 변수, %s", ident.Value)
		}

		c.loadSymbol(symbol)

		err := c.CompileObject(ie.Right)
		if err != nil {
			return err
		}

		c.emit(code.OpAdd)

		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
		}

		return nil
	case "-=":
		ident, ok := ie.Left.(*ast.Identifier)
		if !ok {
			return fmt.Errorf("-= 연산자 왼쪽 값이 변수가 아닙니다. %v", ident)
		}

		symbol, ok := c.symbolTable.Resolve(ident.Value)
		if !ok {
			return fmt.Errorf("정의되지 않은 변수, %s", ident.Value)
		}

		c.loadSymbol(symbol)

		err := c.CompileObject(ie.Right)
		if err != nil {
			return err
		}

		c.emit(code.OpSub)

		if symbol.Scope == GlobalScope {
			c.emit(code.OpSetGlobal, symbol.Index)
		} else {
			c.emit(code.OpSetLocal, symbol.Index)
		}

		return nil
	}

	switch ie.Operator {
	case "<":
		err := c.CompileObject(ie.Right)
		if err != nil {
			return err
		}

		err = c.CompileObject(ie.Left)
		if err != nil {
			return err
		}

		c.emit(code.OpGreaterThan)

		return nil
	case "<=":
		err := c.CompileObject(ie.Right)
		if err != nil {
			return err
		}

		err = c.CompileObject(ie.Left)
		if err != nil {
			return err
		}

		c.emit(code.OpGreaterThanOrEquel)

		return nil
	}

	err := c.CompileObject(ie.Left)
	if err != nil {
		return err
	}

	err = c.CompileObject(ie.Right)
	if err != nil {
		return err
	}

	switch ie.Operator {
	case "+":
		c.emit(code.OpAdd)
	case "-":
		c.emit(code.OpSub)
	case "*":
		c.emit(code.OpMul)
	case "/":
		c.emit(code.OpDiv)
	case "%":
		c.emit(code.OpRem)
	case ">":
		c.emit(code.OpGreaterThan)
	case ">=":
		c.emit(code.OpGreaterThanOrEquel)
	case "==":
		c.emit(code.OpEqual)
	case "!=":
		c.emit(code.OpNotEqual)
	default:
		return fmt.Errorf("정의되지 않은 중위 연산자 [%s]", ie.Operator)
	}

	return nil
}

func (c *Compiler) compObjectIntegerLiteral(il *ast.IntegerLiteral) error {
	c.emit(code.OpConstant, c.addConstant(&object.Integer{Value: il.Value}))
	return nil
}

func (c *Compiler) compObjectBooleanLiteral(bl *ast.BooleanLiteral) error {
	if bl.Value {
		c.emit(code.OpTrue)
	} else {
		c.emit(code.OpFalse)
	}
	return nil
}

func (c *Compiler) compObjectIdentifier(i *ast.Identifier) error {
	symbol, ok := c.symbolTable.Resolve(i.Value)
	if !ok {
		// TODO
		c.emit(code.OpGetGlobal, 9999)
		c.reservationSymbol = append(c.reservationSymbol, struct {
			pos              int
			ReplaceFuncIndex int
			symbol           string
		}{
			len(c.currentInstructions()) - 5,
			len(c.constants),
			i.Value,
		})
		return nil
	}

	c.loadSymbol(symbol)

	return nil
}

func (c *Compiler) compObjectStringLiteral(sl *ast.StringLiteral) error {
	c.emit(code.OpConstant, c.addConstant(&object.String{Value: sl.Value}))
	return nil
}

func (c *Compiler) compObjectArrayLiteral(al *ast.ArrayLiteral) error {
	for _, el := range al.Elements {
		err := c.CompileObject(el)
		if err != nil {
			return err
		}
	}

	c.emit(code.OpArray, len(al.Elements))

	return nil
}

func (c *Compiler) compObjectIndexExpression(ie *ast.IndexExpression) error {
	err := c.CompileObject(ie.Left)
	if err != nil {
		return err
	}

	err = c.CompileObject(ie.Index)
	if err != nil {
		return err
	}

	c.emit(code.OpIndex)

	return nil
}

func (c *Compiler) compObjectCallFunctionExpression(cfe *ast.CallFunctionExpression) error {
	err := c.CompileObject(cfe.Function)
	if err != nil {
		return err
	}

	for _, a := range cfe.Arguments {
		err := c.CompileObject(a)
		if err != nil {
			return err
		}
	}

	c.emit(code.OpCall, len(cfe.Arguments))

	return nil
}
