package evaluator

import (
	"RenG/interpreter/ast"
	"RenG/interpreter/object"
)

func evalForExpression(node *ast.ForExpression, env *object.Environment) object.Object {
	Eval(node.Define, env)
	condition := Eval(node.Condition, env)
	for isTruthy(condition) {
		result := Eval(node.Body, env)
		if _, ok := result.(*object.ReturnValue); ok {
			return result
		}
		Eval(node.Run, env)
		condition = Eval(node.Condition, env)
	}
	return nil
}
