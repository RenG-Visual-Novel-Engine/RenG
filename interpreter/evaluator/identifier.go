package evaluator

import (
	"RenG/interpreter/ast"
	"RenG/interpreter/object"
)

func evalIdentifier(node *ast.Identifier, env *object.Environment) object.Object {
	val, ok := env.Get(node.Value)
	if !ok {
		return newError("identifier not found: " + node.Value)
	}

	return val
}
