package vm

import (
	"RenG/RVM/src/core/System/bytecode"
	"RenG/RVM/src/core/System/object"
	"fmt"
)

type VM struct {
	constants []object.Object

	stack []object.Object
	sp    int

	globals []object.Object

	frames      []Frame
	framesIndex int
}

func NewVM(bytecode *bytecode.Bytecode) *VM {
	mainFn := &object.CompiledFunction{Instructions: bytecode.Instructions}
	mainFrame := NewFrmae(mainFn, 0)

	frames := make([]Frame, 1024)
	frames[0] = mainFrame

	return &VM{
		constants:   bytecode.Constants,
		stack:       make([]object.Object, StackSize),
		sp:          0,
		globals:     make([]object.Object, 65536),
		frames:      frames,
		framesIndex: 1,
	}
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

func (vm *VM) currentFrame() Frame {
	return vm.frames[vm.framesIndex-1]
}

func (vm *VM) pushFrame(f Frame) {
	vm.frames[vm.framesIndex] = f
	vm.framesIndex++
}

func (vm *VM) popFrame() Frame {
	vm.framesIndex--
	return vm.frames[vm.framesIndex]
}
