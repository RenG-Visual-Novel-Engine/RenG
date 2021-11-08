package screen

import (
	"RenG/src/config"
	"RenG/src/core"
	"RenG/src/lang/ast"
	"RenG/src/lang/object"
	"fmt"
	"sync"
)

var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

var (
	ScreenMutex = &sync.RWMutex{}
)

func IsInTexture(texture *core.SDL_Texture, x, y int) bool {
	return x >= texture.Xpos && x <= texture.Width+texture.Xpos && y >= texture.Ypos && y <= texture.Height+texture.Ypos
}

func IsFirstPriority(name string) bool {
	if len(config.ScreenPriority) <= 0 {
		return false
	}
	return config.ScreenPriority[len(config.ScreenPriority)-1] == name
}

func FindScreenPriority(name string) int {
	for i := 0; i < len(config.ScreenPriority); i++ {
		if config.ScreenPriority[i] == name {
			return i
		}
	}

	return -1
}

func IsScreenEnd(name string) bool {
	_, ok := config.ScreenAllIndex[name]
	return !ok
}

func applyFunction(fn object.Object, args []object.Object, name string) object.Object {
	switch fn := fn.(type) {
	case *object.Function:
		extendedEnv := extendFunctionEnv(fn, args)
		evaluated := ScreenEval(fn.Body, extendedEnv, name)
		return unwrapReturnValue(evaluated)
	case *object.Builtin:
		return fn.Fn(args...)
	default:
		return newError("not a function: %s", fn.Type())
	}
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
		return returnValue.Value
	}
	return obj
}

func isCurrentExp(index int, str *ast.StringLiteral) bool {
	for i := 0; i < len(str.Exp); i++ {
		if index == str.Exp[i].Index {
			return true
		} else if index < str.Exp[i].Index {
			return false
		}
	}
	return false
}

func isTruthy(obj object.Object) bool {
	switch obj {
	case NULL:
		return false
	case TRUE:
		return true
	case FALSE:
		return false
	default:
		return true
	}
}

func isAssign(operator string) bool {
	switch operator {
	case "=":
		return true
	case "+=":
		return true
	case "-=":
		return true
	case "*=":
		return true
	case "/=":
		return true
	case "%=":
		return true
	default:
		return false
	}
}

func isError(obj object.Object) bool {
	if obj != nil {
		return obj.Type() == object.ERROR_OBJ
	}
	return false
}

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}
