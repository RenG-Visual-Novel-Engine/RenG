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
	constants []object.Object

	stack []object.Object
	sp    int // 항상 다음 스택을 가리킴

	globals []object.Object

	frames      []*Frame
	framesIndex int
}

func New(bytecode *compiler.Bytecode) *VM {
	mainFn := &object.CompiledFunction{Instructions: bytecode.Instructions}
	mainFrame := NewFrmae(mainFn, 0)

	frames := make([]*Frame, 1024)
	frames[0] = mainFrame

	return &VM{
		constants: bytecode.Constants,

		stack: make([]object.Object, StackSize),
		sp:    0,

		globals: make([]object.Object, 65536),

		frames:      frames,
		framesIndex: 1,
	}
}

func (vm *VM) Run() error {
	var ip int
	var ins code.Instructions
	var op code.Opcode

	for vm.currentFrame().ip < len(vm.currentFrame().Instructions())-1 {
		vm.currentFrame().ip++

		ip = vm.currentFrame().ip
		ins = vm.currentFrame().Instructions()
		op = code.Opcode(ins[ip])

		switch op {
		case code.OpConstant:
			constIndex := code.ReadUint32(ins[ip+1:])
			vm.currentFrame().ip += 4

			err := vm.push(vm.constants[constIndex])
			if err != nil {
				return err
			}
		case code.OpAdd:
			right := vm.pop()
			left := vm.pop()

			// fmt.Println(left)

			if right.Type() == object.INTEGER_OBJ && left.Type() == object.INTEGER_OBJ {
				vm.push(&object.Integer{Value: left.(*object.Integer).Value + right.(*object.Integer).Value})
			} else if right.Type() == object.STRING_OBJ && left.Type() == object.STRING_OBJ {
				vm.push(&object.String{Value: left.(*object.String).Value + right.(*object.String).Value})
			}
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
		case code.OpRem:
			right := vm.pop().(*object.Integer).Value
			left := vm.pop().(*object.Integer).Value

			vm.push(&object.Integer{Value: left % right})
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
		case code.OpEqual, code.OpNotEqual, code.OpGreaterThan, code.OpGreaterThanOrEquel:
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
				case code.OpGreaterThan:
					if left.(*object.Integer).Value > right.(*object.Integer).Value {
						vm.push(True)
					} else {
						vm.push(False)
					}
				case code.OpGreaterThanOrEquel:
					if left.(*object.Integer).Value >= right.(*object.Integer).Value {
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
			} else if left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ {
				switch op {
				case code.OpEqual:
					if right.(*object.String).Value == left.(*object.String).Value {
						vm.push(True)
					} else {
						vm.push(False)
					}
				case code.OpNotEqual:
					if right.(*object.String).Value != left.(*object.String).Value {
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
			pos := int(code.ReadUint16(ins[ip+1:]))
			vm.currentFrame().ip = pos - 1
		case code.OpJumpNotTruthy:
			pos := int(code.ReadUint16(ins[ip+1:]))
			vm.currentFrame().ip += 2

			condition := vm.pop()
			if !isTruthy(condition) {
				vm.currentFrame().ip = pos - 1
			}
		case code.OpSetGlobal:
			globalIndex := code.ReadUint32(ins[ip+1:])
			vm.currentFrame().ip += 4

			vm.globals[globalIndex] = vm.pop()
		case code.OpGetGlobal:
			globalIndex := code.ReadUint32(ins[ip+1:])
			vm.currentFrame().ip += 4

			err := vm.push(vm.globals[globalIndex])
			// fmt.Println(globalIndex)
			if err != nil {
				return err
			}
		case code.OpArray:
			numElement := int(code.ReadUint32(ins[ip+1:]))
			vm.currentFrame().ip += 4

			elements := make([]object.Object, vm.sp-(vm.sp-numElement))

			for i := vm.sp - numElement; i < vm.sp; i++ {
				elements[i-(vm.sp-numElement)] = vm.stack[i]
			}

			array := &object.Array{Elements: elements}
			vm.sp = vm.sp - numElement

			err := vm.push(array)
			if err != nil {
				return err
			}
		case code.OpIndex:
			index := vm.pop()
			left := vm.pop()

			switch left.Type() {
			case object.ARRAY_OBJ:
				array := left.(*object.Array)
				i := index.(*object.Integer).Value
				max := int64(len(array.Elements) - 1)

				if i < 0 || i > max {
					vm.push(Null)
					return nil
				}

				vm.push(array.Elements[i])
			case object.STRING_OBJ:
				str := left.(*object.String)
				i := index.(*object.Integer).Value
				max := int64(len(str.Value) - 1)

				if i < 0 || i > max {
					vm.push(Null)
					return nil
				}

				vm.push(&object.String{Value: string([]rune(str.Value)[i])})
			}
		case code.OpCall:
			numArgs := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2

			callee := vm.stack[vm.sp-1-int(numArgs)]
			// fmt.Println(numArgs)
			switch callee := callee.(type) {
			case *object.CompiledFunction:
				if int(numArgs) != callee.NumParameters {
					return fmt.Errorf("error")
				}

				frame := NewFrmae(callee, vm.sp-int(numArgs))
				vm.pushFrame(frame)

				vm.sp = frame.basePointer + callee.NumLocals
			case *object.Builtin:
				args := vm.stack[vm.sp-int(numArgs) : vm.sp]
				result := callee.Fn(args...)
				vm.sp = vm.sp - int(numArgs) - 1

				if result != nil {
					vm.push(result)
				} else {
					vm.push(Null)
				}
			default:
				// fmt.Println("s")
				return fmt.Errorf("error")
			}
		case code.OpReturnValue:
			returnValue := vm.pop()

			frame := vm.popFrame()
			vm.sp = frame.basePointer - 1

			err := vm.push(returnValue)
			if err != nil {
				return err
			}
		case code.OpReturn:
			frame := vm.popFrame()
			vm.sp = frame.basePointer - 1

			err := vm.push(Null)
			if err != nil {
				return err
			}
		case code.OpSetLocal:
			localIndex := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2

			frame := vm.currentFrame()

			vm.stack[frame.basePointer+int(localIndex)] = vm.pop()
		case code.OpGetLocal:
			localIndex := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2

			frame := vm.currentFrame()

			err := vm.push(vm.stack[frame.basePointer+int(localIndex)])
			if err != nil {
				return err
			}
		case code.OpGetBuiltin:
			builtinIndex := code.ReadUint16(ins[ip+1:])
			vm.currentFrame().ip += 2

			definition := object.FunctionBuiltins[builtinIndex]

			err := vm.push(definition.Builtin)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (vm *VM) push(o object.Object) error {
	if vm.sp >= StackSize {
		return fmt.Errorf("stack overflow")
	}

	// fmt.Printf("push %s\n", o.Inspect())

	vm.stack[vm.sp] = o
	vm.sp++

	return nil
}

func (vm *VM) pop() object.Object {
	o := vm.stack[vm.sp-1]
	// fmt.Printf("pop %s\n", o.Inspect())
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

func (vm *VM) currentFrame() *Frame {
	return vm.frames[vm.framesIndex-1]
}

func (vm *VM) pushFrame(f *Frame) {
	vm.frames[vm.framesIndex] = f
	vm.framesIndex++
}

func (vm *VM) popFrame() *Frame {
	vm.framesIndex--
	return vm.frames[vm.framesIndex]
}
