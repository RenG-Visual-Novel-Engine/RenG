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

var status = false

func unwrapReturnValue(obj object.Object) object.Object {
	if returnValue, ok := obj.(*object.ReturnValue); ok {
		if value, ok := returnValue.Value.(*object.Integer); ok && status {
			fmt.Println(value.Value)
		} else if boolean, ok := returnValue.Value.(*object.Boolean); ok {
			if boolean.Value == false {
				status = false
			} else {
				status = true
			}
		}
		return returnValue.Value
	}
	return obj
}
