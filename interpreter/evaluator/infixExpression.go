package evaluator

import (
	"RenG/interpreter/ast"
	"RenG/interpreter/object"
)

func evalInfixExpression(operator string, left, right object.Object) object.Object {
	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)
	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
	case left.Type() != right.Type():
		return newError("type mismatch: %s %s %s", left.Type(), operator, right.Type())
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalAssignInfixExpression(operator string, left *ast.Identifier, right object.Object, env *object.Environment) {
	switch operator {
	case "=":
		env.Set(left.Value, right)
	case "+=":
		leftVal, _ := env.Get(left.Value)
		env.Set(left.Value, &object.Integer{Value: leftVal.(*object.Integer).Value + right.(*object.Integer).Value})
	case "-=":
		leftVal, _ := env.Get(left.Value)
		env.Set(left.Value, &object.Integer{Value: leftVal.(*object.Integer).Value - right.(*object.Integer).Value})
	case "*=":
		leftVal, _ := env.Get(left.Value)
		env.Set(left.Value, &object.Integer{Value: leftVal.(*object.Integer).Value * right.(*object.Integer).Value})
	case "/=":
		leftVal, _ := env.Get(left.Value)
		env.Set(left.Value, &object.Integer{Value: leftVal.(*object.Integer).Value / right.(*object.Integer).Value})
	case "%=":
		leftVal, _ := env.Get(left.Value)
		env.Set(left.Value, &object.Integer{Value: leftVal.(*object.Integer).Value % right.(*object.Integer).Value})
	}
}

func evalStringInfixExpression(operator string, left, right object.Object) object.Object {
	if operator != "+" {
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}

	leftVal := left.(*object.String).Value
	rightVal := right.(*object.String).Value
	return &object.String{Value: leftVal + rightVal}
}

func evalIntegerInfixExpression(operator string, left, right object.Object) object.Object {
	leftVal := left.(*object.Integer).Value
	rightVal := right.(*object.Integer).Value
	switch operator {
	case "+":
		return &object.Integer{Value: leftVal + rightVal}
	case "-":
		return &object.Integer{Value: leftVal - rightVal}
	case "*":
		return &object.Integer{Value: leftVal * rightVal}
	case "/":
		return &object.Integer{Value: leftVal / rightVal}
	case "%":
		return &object.Integer{Value: leftVal % rightVal}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "<=":
		return nativeBoolToBooleanObject(leftVal <= rightVal)
	case ">=":
		return nativeBoolToBooleanObject(leftVal >= rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}
	return FALSE
}
