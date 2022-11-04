package compiler

import (
	"RenG/Compiler/core/ast"
	"RenG/Compiler/core/code"
	"RenG/Compiler/core/object"
	"fmt"
)

type Compiler struct {
	instructions code.Instructions
	constants    []object.Object

	symbolTable       *SymbolTable
	reservationSymbol []struct {
		pos              int
		ReplaceFuncIndex int
		symbol           string
	}

	scopes     []CompilationScope
	scopeIndex int
}

type EmittedInstruction struct {
	OpCode   code.Opcode
	Position int
}

type CompilationScope struct {
	instructions        code.Instructions
	lastInstruction     EmittedInstruction
	previousInstruction EmittedInstruction
}

func New() *Compiler {
	mainScope := CompilationScope{
		instructions:        code.Instructions{},
		lastInstruction:     EmittedInstruction{},
		previousInstruction: EmittedInstruction{},
	}

	symbolTable := NewSymbolTable()

	for i, v := range object.FunctionBuiltins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	return &Compiler{
		instructions: code.Instructions{},
		constants:    []object.Object{},
		symbolTable:  symbolTable,
		reservationSymbol: []struct {
			pos              int
			ReplaceFuncIndex int
			symbol           string
		}{},
		scopes:     []CompilationScope{mainScope},
		scopeIndex: 0,
	}
}

func (c *Compiler) Compile(node ast.Node) error {
	switch node := node.(type) {
	case *ast.Program:
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				return err
			}
		}
	case *ast.ExpressionStatement:
		err := c.compExpressionStatement(node)
		if err != nil {
			return err
		}
	case *ast.BlockStatement:
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				return err
			}
		}
	case *ast.FunctionStatement:
		symbol := c.symbolTable.Define(node.Name.Value)

		c.enterScope()

		for _, p := range node.Parameters {
			c.symbolTable.Define(p.Value)
		}

		err := c.Compile(node.Body)
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
			NumParameters: len(node.Parameters),
		}

		c.emit(code.OpConstant, c.addConstant(compiledFn))
		c.emit(code.OpSetGlobal, symbol.Index)
		return nil
	case *ast.IfStatement:
		var jmpPos []int

		err := c.Compile(node.Condition)
		if err != nil {
			return err
		}

		jumpNotTruthyPos := c.emit(code.OpJumpNotTruthy, 9999)

		err = c.Compile(node.Consequence)
		if err != nil {
			return err
		}

		if c.lastInsructionIs(code.OpPop) {
			c.removeLastPop()
		}

		if len(node.Elif) == 0 && node.Else == nil {
			afterConsequencePos := len(c.currentInstructions())
			c.changeOperand(jumpNotTruthyPos, afterConsequencePos)
		} else if len(node.Elif) != 0 {

			for _, elif := range node.Elif {
				jmpPos = append(jmpPos, c.emit(code.OpJump, 9999))

				afterConsequencePos := len(c.currentInstructions())
				c.changeOperand(jumpNotTruthyPos, afterConsequencePos)

				err := c.Compile(elif.Condition)
				if err != nil {
					return err
				}

				jumpNotTruthyPos = c.emit(code.OpJumpNotTruthy, 9999)

				err = c.Compile(elif.Consequence)
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

		if node.Else != nil {
			err := c.Compile(node.Else)
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
	case *ast.ForStatement:
		var jmpNotTruthyPos, jmpPos int = -1, -1

		if node.Initialization != nil {
			err := c.Compile(node.Initialization)
			if err != nil {
				return err
			}
		}

		jmpPos = len(c.currentInstructions())

		if node.Condition != nil {
			err := c.Compile(node.Condition)
			if err != nil {
				return err
			}

			jmpNotTruthyPos = c.emit(code.OpJumpNotTruthy, 9999)
		}

		err := c.Compile(node.Loop)
		if err != nil {
			return err
		}

		if node.Increment != nil {
			err := c.Compile(node.Increment)
			if err != nil {
				return err
			}
		}

		c.emit(code.OpJump, jmpPos)

		if jmpNotTruthyPos != -1 {
			c.changeOperand(jmpNotTruthyPos, len(c.currentInstructions()))
		}

	case *ast.ReturnStatement:
		err := c.Compile(node.ReturnValue)
		if err != nil {
			return err
		}

		c.emit(code.OpReturnValue)

	case *ast.PrefixExpression:
		err := c.Compile(node.Right)
		if err != nil {
			return err
		}

		switch node.Operator {
		case "-":
			c.emit(code.OpMinus)
		case "!":
			c.emit(code.OpBang)
		default:
			return fmt.Errorf("unknown operator")
		}
	case *ast.InfixExpression:
		switch node.Operator {
		case "=":
			symbol, ok := c.symbolTable.Resolve(node.Left.TokenLiteral())
			if !ok {
				symbol = c.symbolTable.Define(node.Left.TokenLiteral())
			}

			err := c.Compile(node.Right)
			if err != nil {
				return err
			}
			if symbol.Scope == GlobalScope {
				c.emit(code.OpSetGlobal, symbol.Index)
			} else {
				c.emit(code.OpSetLocal, symbol.Index)
			}
			return nil
		case "<":
			err := c.Compile(node.Right)
			if err != nil {
				return err
			}

			err = c.Compile(node.Left)
			if err != nil {
				return err
			}

			c.emit(code.OpGreaterThan)
			return nil
		case "<=":
			err := c.Compile(node.Right)
			if err != nil {
				return err
			}

			err = c.Compile(node.Left)
			if err != nil {
				return err
			}

			c.emit(code.OpGreaterThanOrEquel)
			return nil
		default:
			err := c.Compile(node.Left)
			if err != nil {
				return err
			}

			err = c.Compile(node.Right)
			if err != nil {
				return err
			}

			switch node.Operator {
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
				return fmt.Errorf("unknown operator %s", node.Operator)
			}
		}
	case *ast.IntegerLiteral:
		integer := &object.Integer{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(integer))
	case *ast.Boolean:
		if node.Value {
			c.emit(code.OpTrue)
		} else {
			c.emit(code.OpFalse)
		}
	case *ast.Identifier:
		symbol, ok := c.symbolTable.Resolve(node.Value)
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
				node.Value,
			})
			return nil
		}

		c.loadSymbol(symbol)

	case *ast.StringLiteral:
		str := &object.String{Value: node.Value}
		c.emit(code.OpConstant, c.addConstant(str))
	case *ast.ArrayLiteral:
		for _, el := range node.Elements {
			err := c.Compile(el)
			if err != nil {
				return err
			}
		}

		c.emit(code.OpArray, len(node.Elements))
	case *ast.IndexExpression:
		err := c.Compile(node.Left)
		if err != nil {
			return err
		}

		err = c.Compile(node.Index)
		if err != nil {
			return err
		}

		c.emit(code.OpIndex)
	case *ast.CallFunctionExpression:
		err := c.Compile(node.Function)
		if err != nil {
			return err
		}

		for _, a := range node.Arguments {
			err := c.Compile(a)
			if err != nil {
				return err
			}
		}

		c.emit(code.OpCall, len(node.Arguments))
	}
	return nil
}
