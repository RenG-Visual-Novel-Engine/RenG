package reng

import (
	"RenG/src/config"
	"RenG/src/core"
	"RenG/src/lang/ast"
	"RenG/src/lang/object"
	"fmt"
)

var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

func playMusic(musicRoot string, loop bool) {
	if !core.PlayingMusic() {
		loadMusic(musicRoot).PlayMusic(loop)
	} else {
		core.StopMusic(-1)
		loadMusic(musicRoot).PlayMusic(loop)
	}
}

func loadMusic(root string) *core.Mix_Music {
	if music, ok := config.MusicList.Get(root); ok {
		return music
	} else {
		music = config.MusicList.Set(root, core.LoadMUS(root))
		return music
	}
}

func play(soundRoot string, channel int) {
	if !core.PlayingMusicChannel(channel) {
		loadChunk(soundRoot).PlaySound(channel)
	} else {
		core.StopMusic(channel)
		loadChunk(soundRoot).PlaySound(channel)
	}
}

func loadChunk(root string) *core.Mix_Chunk {
	if chunk, ok := config.ChunkList.Get(root); ok {
		return chunk
	} else {
		chunk = config.ChunkList.Set(root, core.LoadWAV(root))
		return chunk
	}
}

func applyFunction(fn object.Object, args []object.Object) object.Object {
	switch fn := fn.(type) {
	case *object.Function:
		extendedEnv := extendFunctionEnv(fn, args)
		evaluated := RengEval(fn.Body, extendedEnv)
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
