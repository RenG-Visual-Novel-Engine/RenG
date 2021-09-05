package evaluator

import (
	"RenG/interpreter/ast"
	"RenG/interpreter/object"
)

func evalIfExpression(ie *ast.IfExpression, env *object.Environment) object.Object {
	condition := Eval(ie.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return Eval(ie.Consequence, env)
	}

	for _, ee := range ie.Elif {
		if ee != nil {
			elifCondition := Eval(ee.Condition, env)
			if isError(elifCondition) {
				return elifCondition
			}
			if isTruthy(elifCondition) {
				return Eval(ee.Consequence, env)
			}
		}
	}

	if ie.Alternative != nil {
		return Eval(ie.Alternative, env)
	} else {
		return NULL
	}
}
