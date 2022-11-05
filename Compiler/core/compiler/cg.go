package compiler

import (
	"RenG/Compiler/core/ast"
	"RenG/Compiler/core/code"
	"RenG/Compiler/core/object"
	"fmt"
)

func (c *Compiler) compGlobalProgram(p *ast.Program) error {
	for _, s := range p.Statements {
		err := c.CompileGlobal(s)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Compiler) compGlobalExpressionStatement(es *ast.ExpressionStatement) error {
	err := c.CompileGlobal(es.Expression)
	if err != nil {
		return err
	}

	if !c.lastInsructionIs(code.OpSetGlobal) && !c.lastInsructionIs(code.OpSetLocal) {
		c.emit(code.OpPop)
	}

	return nil
}

func (c *Compiler) compGlobalBlockStatement(bs *ast.BlockStatement) error {
	for _, s := range bs.Statements {
		err := c.CompileGlobal(s)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Compiler) compGlobalFunctionStatement(fs *ast.FunctionStatement) error {
	symbol := c.symbolTable.Define(fs.Name.Value)

	c.enterScope()

	for _, p := range fs.Parameters {
		c.symbolTable.Define(p.Value)
	}

	err := c.CompileGlobal(fs.Body)
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

func (c *Compiler) compGlobalIfStatement(is *ast.IfStatement) error {
	var jmpPos []int

	err := c.CompileGlobal(is.Condition)
	if err != nil {
		return err
	}

	jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

	err = c.CompileGlobal(is.Consequence)
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

			err := c.CompileGlobal(elif.Condition)
			if err != nil {
				return err
			}

			jumpNotTruthyPos = c.emit(code.OpJumpNotTruthy, 9999)

			err = c.CompileGlobal(elif.Consequence)
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
		err := c.CompileGlobal(is.Else)
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

func (c *Compiler) compGlobalForStatement(fs *ast.ForStatement) error {
	var jmpNotTruthyPos, jmpPos int = -1, -1

	if fs.Initialization != nil {
		err := c.CompileGlobal(fs.Initialization)
		if err != nil {
			return err
		}
	}

	jmpPos = len(c.currentInstructions())

	if fs.Condition != nil {
		err := c.CompileGlobal(fs.Condition)
		if err != nil {
			return err
		}

		jmpNotTruthyPos = c.emit(code.OpJumpNotTruthy, 9999)
	}

	err := c.CompileGlobal(fs.Loop)
	if err != nil {
		return err
	}

	if fs.Increment != nil {
		err := c.CompileGlobal(fs.Increment)
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

func (c *Compiler) compGlobalReturnStatement(rs *ast.ReturnStatement) error {
	err := c.CompileGlobal(rs.ReturnValue)
	if err != nil {
		return err
	}

	c.emit(code.OpReturnValue)

	return nil
}

func (c *Compiler) compGlobalPrefixExpression(pe *ast.PrefixExpression) error {
	err := c.CompileGlobal(pe.Right)
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

func (c *Compiler) compGlobalInfixExpression(ie *ast.InfixExpression) error {

	switch ie.Operator {
	case "=":
		symbol, ok := c.symbolTable.Resolve(ie.Left.TokenLiteral())
		if !ok {
			symbol = c.symbolTable.Define(ie.Left.TokenLiteral())
		}

		err := c.CompileGlobal(ie.Right)
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
			return fmt.Errorf("+= 연산자 왼쪽 값이 변수가 아닙니다.")
		}

		symbol, ok := c.symbolTable.Resolve(ident.Value)
		if !ok {
			return fmt.Errorf("정의되지 않은 변수, %s", ident.Value)
		}

		c.loadSymbol(symbol)

		err := c.CompileGlobal(ie.Right)
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
			return fmt.Errorf("-= 연산자 왼쪽 값이 변수가 아닙니다.")
		}

		symbol, ok := c.symbolTable.Resolve(ident.Value)
		if !ok {
			return fmt.Errorf("정의되지 않은 변수, %s", ident.Value)
		}

		c.loadSymbol(symbol)

		err := c.CompileGlobal(ie.Right)
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
		err := c.CompileGlobal(ie.Right)
		if err != nil {
			return err
		}

		err = c.CompileGlobal(ie.Left)
		if err != nil {
			return err
		}

		c.emit(code.OpGreaterThan)

		return nil
	case "<=":
		err := c.CompileGlobal(ie.Right)
		if err != nil {
			return err
		}

		err = c.CompileGlobal(ie.Left)
		if err != nil {
			return err
		}

		c.emit(code.OpGreaterThanOrEquel)

		return nil
	}

	err := c.CompileGlobal(ie.Left)
	if err != nil {
		return err
	}

	err = c.CompileGlobal(ie.Right)
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

func (c *Compiler) compGlobalIntegerLiteral(il *ast.IntegerLiteral) error {
	c.emit(code.OpConstant, c.addConstant(&object.Integer{Value: il.Value}))
	return nil
}

func (c *Compiler) compGlobalBooleanLiteral(bl *ast.BooleanLiteral) error {
	if bl.Value {
		c.emit(code.OpTrue)
	} else {
		c.emit(code.OpFalse)
	}
	return nil
}

func (c *Compiler) compGlobalIdentifier(i *ast.Identifier) error {
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

func (c *Compiler) compGlobalStringLiteral(sl *ast.StringLiteral) error {
	c.emit(code.OpConstant, c.addConstant(&object.String{Value: sl.Value}))
	return nil
}

func (c *Compiler) compGlobalArrayLiteral(al *ast.ArrayLiteral) error {
	for _, el := range al.Elements {
		err := c.CompileGlobal(el)
		if err != nil {
			return err
		}
	}

	c.emit(code.OpArray, len(al.Elements))

	return nil
}

func (c *Compiler) compGlobalIndexExpression(ie *ast.IndexExpression) error {
	err := c.CompileGlobal(ie.Left)
	if err != nil {
		return err
	}

	err = c.CompileGlobal(ie.Index)
	if err != nil {
		return err
	}

	c.emit(code.OpIndex)

	return nil
}

func (c *Compiler) compGlobalCallFunctionExpression(cfe *ast.CallFunctionExpression) error {
	err := c.CompileGlobal(cfe.Function)
	if err != nil {
		return err
	}

	for _, a := range cfe.Arguments {
		err := c.CompileGlobal(a)
		if err != nil {
			return err
		}
	}

	c.emit(code.OpCall, len(cfe.Arguments))

	return nil
}
