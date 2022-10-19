package vm

import (
	"RenG/RVM/src/core/System/code"
	"RenG/RVM/src/core/System/object"
	"fmt"
)

func (vm *VM) runOpConstant(ins code.Instructions, ip int) error {
	constIndex := code.ReadUint32(ins[ip+1:])
	vm.currentFrame().SetIp(ip + 4)

	err := vm.push(vm.constants[constIndex])
	if err != nil {
		return err
	}
	return nil
}

func (vm *VM) runOpJumpNotTruthy(ins code.Instructions, ip int) error {
	pos := int(code.ReadUint16(ins[ip+1:]))
	vm.currentFrame().SetIp(ip + 2)

	condition := vm.pop()
	if condition == nil {
		return fmt.Errorf("NullError : OpJumpNotTruthy -> condition - null")
	}
	if !isTruthy(condition) {
		vm.currentFrame().SetIp(pos - 1)
	}

	return nil
}

func (vm *VM) runOpJump(ins code.Instructions, ip int) {
	pos := int(code.ReadUint16(ins[ip+1:]))
	vm.currentFrame().SetIp(pos - 1)
}

func (vm *VM) runOpAdd() error {
	right := vm.pop()
	if right == nil {
		return fmt.Errorf("NullError : (+) 연산자 오른쪽 값 - null")
	}
	left := vm.pop()
	if left == nil {
		return fmt.Errorf("NullError : (+) 연산자 왼쪽 값 - null")
	}

	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		vm.push(&object.Integer{Value: left.(*object.Integer).Value + right.(*object.Integer).Value})
	case left.Type() == object.FLOAT_OBJ && right.Type() == object.FLOAT_OBJ:
		vm.push(&object.Float{Value: left.(*object.Float).Value + right.(*object.Float).Value})
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		vm.push(&object.String{Value: left.(*object.String).Value + right.(*object.String).Value})
	default:
		return fmt.Errorf("TypeError : (+) 연산자에 잘못된 타입이 들어옴 || left : %s - right : %s", left.Type(), right.Type())
	}

	return nil
}

func (vm *VM) runOpSub() error {
	right := vm.pop()
	if right == nil {
		return fmt.Errorf("NullError : (-) 연산자 오른쪽 값 - null")
	}
	left := vm.pop()
	if left == nil {
		return fmt.Errorf("NullError : (-) 연산자 왼쪽 값 - null")
	}

	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		vm.push(&object.Integer{Value: left.(*object.Integer).Value - right.(*object.Integer).Value})
	case left.Type() == object.FLOAT_OBJ && right.Type() == object.FLOAT_OBJ:
		vm.push(&object.Float{Value: left.(*object.Float).Value - right.(*object.Float).Value})
	default:
		return fmt.Errorf("TypeError : (-) 연산자에 잘못된 타입이 들어옴 || left : %s - right : %s", left.Type(), right.Type())
	}

	return nil
}

func (vm *VM) runOpMul() error {
	right := vm.pop()
	if right == nil {
		return fmt.Errorf("NullError : (*) 연산자 오른쪽 값 - null")
	}
	left := vm.pop()
	if left == nil {
		return fmt.Errorf("NullError : (*) 연산자 왼쪽 값 - null")
	}

	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		vm.push(&object.Integer{Value: left.(*object.Integer).Value * right.(*object.Integer).Value})
	case left.Type() == object.FLOAT_OBJ && right.Type() == object.FLOAT_OBJ:
		vm.push(&object.Float{Value: left.(*object.Float).Value * right.(*object.Float).Value})
	default:
		return fmt.Errorf("TypeError : (*) 연산자에 잘못된 타입이 들어옴 || left : %s - right : %s", left.Type(), right.Type())
	}

	return nil
}

func (vm *VM) runOpDiv() error {
	right := vm.pop()
	if right == nil {
		return fmt.Errorf("NullError : (/) 연산자 오른쪽 값 - null")
	}
	left := vm.pop()
	if left == nil {
		return fmt.Errorf("NullError : (/) 연산자 왼쪽 값 - null")
	}

	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		vm.push(&object.Integer{Value: left.(*object.Integer).Value / right.(*object.Integer).Value})
	case left.Type() == object.FLOAT_OBJ && right.Type() == object.FLOAT_OBJ:
		vm.push(&object.Float{Value: left.(*object.Float).Value / right.(*object.Float).Value})
	default:
		return fmt.Errorf("TypeError : (/) 연산자에 잘못된 타입이 들어옴 || left : %s - right : %s", left.Type(), right.Type())
	}

	return nil
}

func (vm *VM) runOpRem() error {
	right := vm.pop()
	if right == nil {
		return fmt.Errorf("NullError : %%) 연산자 오른쪽 값 - null")
	}
	left := vm.pop()
	if left == nil {
		return fmt.Errorf("NullError : (%%) 연산자 왼쪽 값 - null")
	}

	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		vm.push(&object.Integer{Value: left.(*object.Integer).Value % right.(*object.Integer).Value})
	default:
		return fmt.Errorf("TypeError : (%%) 연산자에 잘못된 타입이 들어옴 || left : %s - right : %s", left.Type(), right.Type())
	}

	return nil
}
