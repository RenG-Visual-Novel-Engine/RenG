package evaluator

import (
	"RenG/src/ast"
	"RenG/src/object"
	"fmt"
)

func evalLabelExpression(le *ast.LabelExpression, env *object.Environment) {
	obj := &object.Label{Name: le.Name, Body: le.Body, Env: env}
	env.Set(le.Name.String(), obj)
	if obj.Name.Value == "start" {
		fmt.Println(evalLabelStart())
		return
	}
}

func evalLabelStart() string {
	return "start"
}
