package evaluator

import (
	"RenG/interpreter/ast"
	"RenG/interpreter/object"
)

func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(right)
	default:
		return newError("unknown operator: %s%s", operator, right.Type())
	}
}

func evalAssignPrefixExpression(operator string, right *ast.Identifier, env *object.Environment) object.Object {
	switch operator {
	case "++":
		return evalAssignPrefixPLUS_PLUSExpression(right, env)
	case "--":
		return evalAssignPrefixMINUS_MINUSExpression(right, env)
	default:
		return newError("unknown operator: %s", operator)
	}
}

func evalBangOperatorExpression(right object.Object) object.Object {
	switch right {
	case TRUE:
		return FALSE
	case FALSE:
		return TRUE
	case NULL:
		return FALSE
	default:
		return FALSE
	}
}

func evalMinusPrefixOperatorExpression(right object.Object) object.Object {
	if right.Type() != object.INTEGER_OBJ {
		return newError("unknown operator: -%s", right.Type())
	}
	value := right.(*object.Integer).Value
	return &object.Integer{Value: -value}
}

func evalAssignPrefixPLUS_PLUSExpression(right *ast.Identifier, env *object.Environment) object.Object {
	rightVal, ok := env.Get(right.Value)
	if !ok {
		return newError("Ident has not Value")
	}
	result := &object.Integer{Value: rightVal.(*object.Integer).Value + 1}
	env.Set(right.Value, result)
	return result
}

func evalAssignPrefixMINUS_MINUSExpression(right *ast.Identifier, env *object.Environment) object.Object {
	rightVal, ok := env.Get(right.Value)
	if !ok {
		return newError("Ident has not Value")
	}
	result := &object.Integer{Value: rightVal.(*object.Integer).Value - 1}
	env.Set(right.Value, result)
	return result
}
