package evaluator

import (
	"RenG/interpreter/ast"
	"RenG/interpreter/object"
)

func evalWhileExpression(node *ast.WhileExpression, env *object.Environment) {
	condition := Eval(node.Condition, env)

	for isTruthy(condition) {
		Eval(node.Body, env)
		condition = Eval(node.Condition, env)
	}
}
