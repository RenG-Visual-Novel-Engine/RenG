package evaluator

import (
	"RenG/interpreter/object"
	"fmt"
)

func applyFunction(def object.Object, args []object.Object) object.Object {
	function, ok := def.(*object.Function)
	if !ok {
		return newError("not a function: %s", def.Type())
	}

	extendedEnv := extendFunctionEnv(function, args)
	evaluated := Eval(function.Body, extendedEnv)
	return unwrapReturnValue(evaluated)
}

func extendFunctionEnv(def *object.Function, args []object.Object) *object.Environment {
	env := object.NewEncloseEnvironment(def.Env)

	for paramIdx, param := range def.Parameters {
		env.Set(param.Value, args[paramIdx])
	}

	return env
}

func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		fmt.Println(returnValue.Value.(*object.Integer).Value)
		return returnValue.Value
	}
	return obj
}
