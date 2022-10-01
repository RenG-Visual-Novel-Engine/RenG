package vm

import (
	"RenG/Compiler/code"
	"RenG/Compiler/compiler"
	"RenG/Compiler/object"
	"fmt"
)

const StackSize = 10240

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
