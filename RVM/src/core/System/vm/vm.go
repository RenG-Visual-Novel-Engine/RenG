package vm

import (
	"RenG/RVM/src/core/System/bytecode"
	"RenG/RVM/src/core/System/code"
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

func (vm *VM) Run() error {
	for vm.currentFrame().GetIp() < len(vm.currentFrame().Instructions())-1 {
		vm.currentFrame().SetIp(vm.currentFrame().GetIp() + 1)
		op := code.Opcode(vm.currentFrame().Instructions()[vm.currentFrame().GetIp()])
		switch op {
		case code.OpConstant:
			err := vm.runOpConstant(vm.currentFrame().Instructions(), vm.currentFrame().GetIp())
			if err != nil {
				return err
			}
		case code.OpPop:
			vm.pop()
		case code.OpJumpNotTruthy:
			err := vm.runOpJumpNotTruthy(vm.currentFrame().Instructions(), vm.currentFrame().GetIp())
			if err != nil {
				return err
			}
		case code.OpJump:
			vm.runOpJump(vm.currentFrame().Instructions(), vm.currentFrame().GetIp())
		case code.OpAdd:
			err := vm.runOpAdd()
			if err != nil {
				return err
			}
		case code.OpSub:
			err := vm.runOpSub()
			if err != nil {
				return err
			}
		case code.OpMul:
			err := vm.runOpMul()
			if err != nil {
				return err
			}
		case code.OpDiv:
			err := vm.runOpDiv()
			if err != nil {
				return err
			}
		case code.OpRem:
			err := vm.runOpRem()
			if err != nil {
				return err
			}
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
		case code.OpEqual, code.OpNotEqual, code.OpGreaterThan, code.OpGreaterThanOrEquel:
		case code.OpMinus:
		case code.OpBang:
		case code.OpGetGlobal:
		case code.OpSetGlobal:
		case code.OpArray:
		case code.OpIndex:
		case code.OpCall:
		case code.OpReturn:
		case code.OpReturnValue:
		case code.OpGetLocal:
		case code.OpSetLocal:
		case code.OpGetBuiltin:
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
