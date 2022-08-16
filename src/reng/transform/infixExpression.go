package transform

import (
	"RenG/src/lang/ast"
	"RenG/src/lang/object"
)

func evalInfixExpression(operator string, left, right object.Object) object.Object {
	switch {
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		return evalIntegerInfixExpression(operator, left, right)
	case left.Type() == object.FLOAT_OBJ && right.Type() == object.FLOAT_OBJ:
		return evalFloatInfixExpression(operator, left, right)
	case left.Type() == object.STRING_OBJ && right.Type() == object.STRING_OBJ:
		return evalStringInfixExpression(operator, left, right)
	case left.Type() == object.BOOLEAN_OBJ && right.Type() == object.BOOLEAN_OBJ:
		return evalBooleanInfixExpression(operator, left, right)
	case left.Type() != right.Type():
		return newError("type mismatch: %s %s %s", left.Type(), operator, right.Type())
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalBooleanInfixExpression(operator string, left, right object.Object) object.Object {
	leftVal := left.(*object.Boolean).Value
	rightVal := right.(*object.Boolean).Value

	switch operator {
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	case "&&":
		return nativeBoolToBooleanObject(leftVal && rightVal)
	case "||":
		return nativeBoolToBooleanObject(leftVal || rightVal)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalIntegerAssignInfixExpression(operator string, left *ast.Identifier, leftVal, right object.Object, env *object.Environment) object.Object {
	switch operator {
	case "+=":
		env.Set(left.Value, &object.Integer{Value: leftVal.(*object.Integer).Value + right.(*object.Integer).Value})
	case "-=":
		env.Set(left.Value, &object.Integer{Value: leftVal.(*object.Integer).Value - right.(*object.Integer).Value})
	case "*=":
		env.Set(left.Value, &object.Integer{Value: leftVal.(*object.Integer).Value * right.(*object.Integer).Value})
	case "/=":
		env.Set(left.Value, &object.Integer{Value: leftVal.(*object.Integer).Value / right.(*object.Integer).Value})
	case "%=":
		env.Set(left.Value, &object.Integer{Value: leftVal.(*object.Integer).Value % right.(*object.Integer).Value})
	default:
		return newError("unknown operator: %s %s %s", left.Value, operator, right.Type())
	}
	return nil
}

func evalFloatAssignInfixExpression(operator string, left *ast.Identifier, leftVal, right object.Object, env *object.Environment) object.Object {
	switch operator {
	case "+=":
		env.Set(left.Value, &object.Float{Value: leftVal.(*object.Float).Value + right.(*object.Float).Value})
	case "-=":
		env.Set(left.Value, &object.Float{Value: leftVal.(*object.Float).Value - right.(*object.Float).Value})
	case "*=":
		env.Set(left.Value, &object.Float{Value: leftVal.(*object.Float).Value * right.(*object.Float).Value})
	case "/=":
		env.Set(left.Value, &object.Float{Value: leftVal.(*object.Float).Value / right.(*object.Float).Value})
	default:
		return newError("unknown operator: %s %s %s", left.Value, operator, right.Type())
	}
	return nil
}

func evalAssignInfixExpression(operator string, left *ast.Identifier, right object.Object, env *object.Environment) object.Object {
	switch operator {
	case "=":
		env.Set(left.Value, right)
		return nil
	}

	if val, ok := env.Get(left.Value); ok {
		switch {
		case val.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
			evalIntegerAssignInfixExpression(operator, left, val, right, env)
		case val.Type() == object.FLOAT_OBJ && right.Type() == object.FLOAT_OBJ:
			evalFloatAssignInfixExpression(operator, left, val, right, env)
		}
	}

	return nil
}

func evalStringInfixExpression(operator string, left, right object.Object) object.Object {
	leftVal := left.(*object.String).Value
	rightVal := right.(*object.String).Value

	switch operator {
	case "+":
		return &object.String{Value: leftVal + rightVal}
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return newError("unknown operator: %s %s %s", left.Type(), operator, right.Type())
	}
}

func evalFloatInfixExpression(operator string, left, right object.Object) object.Object {
	leftVal := left.(*object.Float).Value
	rightVal := right.(*object.Float).Value
	switch operator {
	case "+":
		return &object.Float{Value: leftVal + rightVal}
	case "-":
		return &object.Float{Value: leftVal - rightVal}
	case "*":
		return &object.Float{Value: leftVal * rightVal}
	case "/":
		return &object.Float{Value: leftVal / rightVal}
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
	case "&":
		return &object.Integer{Value: leftVal & rightVal}
	case "|":
		return &object.Integer{Value: leftVal | rightVal}
	case "^":
		return &object.Integer{Value: leftVal ^ rightVal}
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
		return object.TRUE
	}
	return object.FALSE
}
