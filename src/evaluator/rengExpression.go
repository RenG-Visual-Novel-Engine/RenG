package evaluator

import (
	"RenG/src/ast"
	"RenG/src/object"
)

func evalLabelExpression(le *ast.LabelExpression, env *object.Environment) {
	obj := &object.Label{Name: le.Name, Body: le.Body}
	env.Set(le.Name.String(), obj)
}
