package compiler

import (
	"RenG/Compiler/ast"
	"RenG/Compiler/code"
	"RenG/Compiler/object"
	"fmt"
)

type Compiler struct {
	instructions code.Instructions
	constants    []object.Object

	lastInsruction      EmittedInstruction
	PreviousInstruction EmittedInstruction
}

type EmittedInstruction struct {
	OpCode   code.Opcode
	Position int
}

func New() *Compiler {
	return &Compiler{
		instructions:        code.Instructions{},
		constants:           []object.Object{},
		lastInsruction:      EmittedInstruction{},
		PreviousInstruction: EmittedInstruction{},
	}
}

func (c *Compiler) Set(ins code.Instructions, con []object.Object) {
	c.instructions = append(c.instructions, ins...)
	c.constants = append(c.constants, con...)
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
		err := c.Compile(node.Expression)
		if err != nil {
			return err
		}
		c.emit(code.OpPop)
	case *ast.BlockStatement:
		for _, s := range node.Statements {
			err := c.Compile(s)
			if err != nil {
				return err
			}
		}
		c.emit(code.OpPop)
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

		if c.lastInsructionIsPop() {
			c.removeLastPop()
		}

		if len(node.Elif) == 0 && node.Else == nil {
			afterConsequencePos := len(c.instructions)
			c.changeOperand(jumpNotTruthyPos, afterConsequencePos)
		} else if len(node.Elif) != 0 {

			for _, elif := range node.Elif {
				jmpPos = append(jmpPos, c.emit(code.OpJump, 9999))

				afterConsequencePos := len(c.instructions)
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

				if c.lastInsructionIsPop() {
					c.removeLastPop()
				}
			}
		}

		jmpPos = append(jmpPos, c.emit(code.OpJump, 9999))

		afterConsequencePos := len(c.instructions)
		c.changeOperand(jumpNotTruthyPos, afterConsequencePos)

		if node.Else != nil {
			err := c.Compile(node.Else)
			if err != nil {
				return err
			}

			if c.lastInsructionIsPop() {
				c.removeLastPop()
			}
		} else {
			c.emit(code.OpNull)
			c.emit(code.OpPop)
		}

		afterElsePos := len(c.instructions)
		for _, jmp := range jmpPos {
			c.changeOperand(jmp, afterElsePos)
		}

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
		case "==":
			c.emit(code.OpEqual)
		case "!=":
			c.emit(code.OpNotEqual)
		default:
			return fmt.Errorf("unknown operator %s", node.Operator)
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
	}
	return nil
}

type Bytecode struct {
	Instructions code.Instructions
	Constants    []object.Object
}

func (c *Compiler) Bytecode() *Bytecode {
	return &Bytecode{
		Instructions: c.instructions,
		Constants:    c.constants,
	}
}

func (c *Compiler) emit(op code.Opcode, operands ...int) int {
	ins := code.Make(op, operands...)
	pos := c.addInstruction(ins)

	c.setLastInstruction(op, pos)

	return pos
}

func (c *Compiler) addConstant(obj object.Object) int {
	c.constants = append(c.constants, obj)
	return len(c.constants) - 1
}

func (c *Compiler) addInstruction(ins []byte) int {
	posNewInstruction := len(c.instructions)
	c.instructions = append(c.instructions, ins...)
	return posNewInstruction
}

func (c *Compiler) setLastInstruction(op code.Opcode, pos int) {
	previous := c.lastInsruction
	last := EmittedInstruction{OpCode: op, Position: pos}

	c.PreviousInstruction = previous
	c.lastInsruction = last
}

func (c *Compiler) lastInsructionIsPop() bool {
	return c.lastInsruction.OpCode == code.OpPop
}

func (c *Compiler) removeLastPop() {
	c.instructions = c.instructions[:c.lastInsruction.Position]
	c.lastInsruction = c.PreviousInstruction
}

func (c *Compiler) replaceInstruction(pos int, newInstruction []byte) {
	for i := 0; i < len(newInstruction); i++ {
		c.instructions[pos+i] = newInstruction[i]
	}
}

func (c *Compiler) changeOperand(opPos int, operand int) {
	op := code.Opcode(c.instructions[opPos])
	newInstruction := code.Make(op, operand)

	c.replaceInstruction(opPos, newInstruction)
}
