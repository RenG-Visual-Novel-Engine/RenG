package evaluator

import (
	"RenG/src/ast"
	"RenG/src/object"
)

func evalLabelExpression(le *ast.LabelExpression, env *object.Environment) object.Object {
	env.Set(le.Name.String(), &object.Label{Name: le.Name, Body: le.Body})

	return nil
}

func evalImageExpression(ie *ast.ImageExpression, env *object.Environment) object.Object {
	rootObj := Eval(ie.Path, env)

	if root, ok := rootObj.(*object.String); ok {
		env.Set(ie.Name.String(), &object.Image{Name: ie.Name, Root: root})
	}

	return nil
}
