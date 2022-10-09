package vm

import (
	"RenG/Compiler/code"
	"RenG/Compiler/compiler"
	"RenG/Compiler/object"
	"fmt"
)

const StackSize = 10240

var True = &object.Boolean{Value: true}
var False = &object.Boolean{Value: false}
var Null = &object.Null{}

type VM struct {
	constants    []object.Object
	instructions code.Instructions

	stack []object.Object
	sp    int // 항상 다음 스택을 가리킴
}

func New(bytecode *compiler.Bytecode) *VM {
	return &VM{
		instructions: bytecode.Instructions,
		constants:    bytecode.Constants,

		stack: make([]object.Object, StackSize),
		sp:    0,
	}
}

func (vm *VM) StackTop() object.Object {
	if vm.sp == 0 {
		return nil
	}
	return vm.stack[vm.sp-1]
}

func (vm *VM) Run() error {
	for ip := 0; ip < len(vm.instructions); ip++ {
		op := code.Opcode(vm.instructions[ip])

		switch op {
		case code.OpConstant:
			constIndex := code.ReadUint32(vm.instructions[ip+1:])
			ip += 4
			err := vm.push(vm.constants[constIndex])
			if err != nil {
				return nil
			}
		case code.OpAdd:
			right := vm.pop().(*object.Integer).Value
			left := vm.pop().(*object.Integer).Value

			vm.push(&object.Integer{Value: left + right})
		case code.OpSub:
			right := vm.pop().(*object.Integer).Value
			left := vm.pop().(*object.Integer).Value

			vm.push(&object.Integer{Value: left - right})
		case code.OpMul:
			right := vm.pop().(*object.Integer).Value
			left := vm.pop().(*object.Integer).Value

			vm.push(&object.Integer{Value: left * right})
		case code.OpDiv:
			right := vm.pop().(*object.Integer).Value
			left := vm.pop().(*object.Integer).Value

			vm.push(&object.Integer{Value: left / right})
		case code.OpPop:
			vm.pop()
		case code.OpTrue:
			err := vm.push(True)
			if err != nil {
				return err
			}
		case code.OpFalse:
			err := vm.push(False)
			if err != nil {
				return err
			}
		case code.OpNull:
			err := vm.push(Null)
			if err != nil {
				return err
			}
		case code.OpEqual, code.OpNotEqual:
			right := vm.pop()
			left := vm.pop()

			if left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ {
				switch op {
				case code.OpEqual:
					if right.(*object.Integer).Value == left.(*object.Integer).Value {
						vm.push(True)
					} else {
						vm.push(False)
					}
				case code.OpNotEqual:
					if right.(*object.Integer).Value != left.(*object.Integer).Value {
						vm.push(True)
					} else {
						vm.push(False)
					}
				}
			} else if left.Type() == object.BOOLEAN_OBJ && right.Type() == object.BOOLEAN_OBJ {
				switch op {
				case code.OpEqual:
					if right.(*object.Boolean).Value == left.(*object.Boolean).Value {
						vm.push(True)
					} else {
						vm.push(False)
					}
				case code.OpNotEqual:
					if right.(*object.Boolean).Value != left.(*object.Boolean).Value {
						vm.push(True)
					} else {
						vm.push(False)
					}
				}
			}
		case code.OpBang:
			operand := vm.pop()

			switch operand {
			case True:
				vm.push(False)
			case False:
				vm.push(True)
			case Null:
				vm.push(Null)
			default:
				vm.push(False)
			}
		case code.OpMinus:
			operand := vm.pop()

			vm.push(&object.Integer{Value: -operand.(*object.Integer).Value})

		case code.OpJump:
			pos := int(code.ReadUint16(vm.instructions[ip+1:]))
			ip = pos - 1
		case code.OpJumpNotTruthy:
			pos := int(code.ReadUint16(vm.instructions[ip+1:]))
			ip += 2

			condition := vm.pop()
			if !isTruthy(condition) {
				ip = pos - 1
			}

		}
	}

	return nil
}

func (vm *VM) push(o object.Object) error {
	if vm.sp >= StackSize {
		return fmt.Errorf("stack overflow")
	}

	vm.stack[vm.sp] = o
	vm.sp++

	return nil
}

func (vm *VM) pop() object.Object {
	o := vm.stack[vm.sp-1]
	vm.sp--
	return o
}

func (vm *VM) LastPoppedStackElem() object.Object {
	return vm.stack[vm.sp]
}

func isTruthy(obj object.Object) bool {
	switch obj := obj.(type) {
	case *object.Boolean:
		return obj.Value
	case *object.Null:
		return false
	default:
		return true
	}
}
