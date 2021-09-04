package evaluator

import (
	"RenG/interpreter/ast"
	"RenG/interpreter/object"
)

func evalFuntionExpression(ie *ast.FunctionExpression, env *object.Environment) {
	obj := &object.Function{Parameters: ie.Parameters, Env: env, Body: ie.Body, Name: ie.Name}
	env.Set(ie.Name.String(), obj)
}
