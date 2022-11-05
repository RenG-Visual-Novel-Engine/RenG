package compiler

import (
	"RenG/Compiler/core/ast"
	"RenG/Compiler/core/code"
	"RenG/Compiler/core/object"
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

func (c *Compiler) CompileObject(node ast.Node) error {
	switch node := node.(type) {
	case *ast.Program:
		err := c.compObjectProgram(node)
		if err != nil {
			return err
		}
	case *ast.ExpressionStatement:
		err := c.compObjectExpressionStatement(node)
		if err != nil {
			return err
		}
	case *ast.BlockStatement:
		err := c.compObjectBlockStatement(node)
		if err != nil {
			return err
		}
	case *ast.FunctionStatement:
		err := c.compObjectFunctionStatement(node)
		if err != nil {
			return err
		}
	case *ast.IfStatement:
		err := c.compObjectIfStatement(node)
		if err != nil {
			return err
		}
	case *ast.ForStatement:
		err := c.compObjectForStatement(node)
		if err != nil {
			return err
		}
	case *ast.ReturnStatement:
		err := c.compObjectReturnStatement(node)
		if err != nil {
			return err
		}
	case *ast.PrefixExpression:
		err := c.compObjectPrefixExpression(node)
		if err != nil {
			return err
		}
	case *ast.InfixExpression:
		err := c.compObjectInfixExpression(node)
		if err != nil {
			return err
		}
	case *ast.IntegerLiteral:
		err := c.compObjectIntegerLiteral(node)
		if err != nil {
			return err
		}
	case *ast.BooleanLiteral:
		err := c.compObjectBooleanLiteral(node)
		if err != nil {
			return err
		}
	case *ast.Identifier:
		err := c.compObjectIdentifier(node)
		if err != nil {
			return err
		}
	case *ast.StringLiteral:
		err := c.compObjectStringLiteral(node)
		if err != nil {
			return err
		}
	case *ast.ArrayLiteral:
		err := c.compObjectArrayLiteral(node)
		if err != nil {
			return err
		}
	case *ast.IndexExpression:
		err := c.compObjectIndexExpression(node)
		if err != nil {
			return err
		}
	case *ast.CallFunctionExpression:
		err := c.compObjectCallFunctionExpression(node)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Compiler) CompileGlobal(node ast.Node) error {
	switch node := node.(type) {
	case *ast.Program:
		err := c.compGlobalProgram(node)
		if err != nil {
			return err
		}
	case *ast.ExpressionStatement:
		err := c.compGlobalExpressionStatement(node)
		if err != nil {
			return err
		}
	case *ast.BlockStatement:
		err := c.compGlobalBlockStatement(node)
		if err != nil {
			return err
		}
	case *ast.FunctionStatement:
		err := c.compGlobalFunctionStatement(node)
		if err != nil {
			return err
		}
	case *ast.IfStatement:
		err := c.compGlobalIfStatement(node)
		if err != nil {
			return err
		}
	case *ast.ForStatement:
		err := c.compGlobalForStatement(node)
		if err != nil {
			return err
		}
	case *ast.ReturnStatement:
		err := c.compGlobalReturnStatement(node)
		if err != nil {
			return err
		}
	case *ast.PrefixExpression:
		err := c.compGlobalPrefixExpression(node)
		if err != nil {
			return err
		}
	case *ast.InfixExpression:
		err := c.compGlobalInfixExpression(node)
		if err != nil {
			return err
		}
	case *ast.IntegerLiteral:
		err := c.compGlobalIntegerLiteral(node)
		if err != nil {
			return err
		}
	case *ast.BooleanLiteral:
		err := c.compGlobalBooleanLiteral(node)
		if err != nil {
			return err
		}
	case *ast.Identifier:
		err := c.compGlobalIdentifier(node)
		if err != nil {
			return err
		}
	case *ast.StringLiteral:
		err := c.compGlobalStringLiteral(node)
		if err != nil {
			return err
		}
	case *ast.ArrayLiteral:
		err := c.compGlobalArrayLiteral(node)
		if err != nil {
			return err
		}
	case *ast.IndexExpression:
		err := c.compGlobalIndexExpression(node)
		if err != nil {
			return err
		}
	case *ast.CallFunctionExpression:
		err := c.compGlobalCallFunctionExpression(node)
		if err != nil {
			return err
		}
	}
	return nil
}
