package screen

import (
	"RenG/src/config"
	"RenG/src/core"
	"RenG/src/lang/ast"
	"RenG/src/lang/object"
	"RenG/src/reng/style"
	"RenG/src/reng/transform"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

var (
	ScreenMutex = &sync.RWMutex{}
)

func typingEffect(te *ast.TextExpression, env *object.Environment, width uint, name, text string) {
	trans, _ := env.Get(te.Transform.Value)
	sty, _ := env.Get(te.Style.Value)

	var (
		textTexture     *core.SDL_Texture
		textTextureList []*core.SDL_Texture
	)

	if int(text[0]) > 126 {
		textTexture = config.MainFont.LoadFromRenderedText(
			text[0:3],
			config.Renderer,
			width,
			core.CreateColor(0xFF, 0xFF, 0xFF),
			255,
			0,
		)
	} else {
		textTexture = config.MainFont.LoadFromRenderedText(
			text[0:1],
			config.Renderer,
			width,
			core.CreateColor(0xFF, 0xFF, 0xFF),
			255,
			0,
		)
	}

	if trans != nil {
		transform.TransformEval(trans.(*object.Transform).Body, textTexture, env)
	} else {
		transform.TransformEval(
			transform.BuiltInsTransform("default", textTexture.TextTexture.Width, textTexture.TextTexture.Height),
			textTexture, env,
		)
	}

	if sty != nil {
		style.StyleEval(sty.(*object.Style).Body, textTexture, env)
	}

	textTextureList = append(textTextureList, textTexture)

	config.ScreenAllIndex[name] = config.Screen{
		First: config.ScreenAllIndex[name].First,
		Count: config.ScreenAllIndex[name].Count + 1,
	}
	config.AddScreenTextureIndex(textTexture)
	indexScreenTexture := len(config.ScreenTextureIndex) - 1

	config.LayerMutex.Lock()
	config.LayerList.Layers[2].AddNewTexture(textTexture)
	indexLayer := len(config.LayerList.Layers[2].Images) - 1
	config.LayerMutex.Unlock()

	for i := 1; i < len(text)+1; i++ {
		if int(text[i-1]) > 126 {
			i += 2
		}

		// TODO : 생성된 텍스쳐 Detroy 필요
		textTexture = config.MainFont.LoadFromRenderedText(
			text[0:i],
			config.Renderer,
			width,
			core.CreateColor(0xFF, 0xFF, 0xFF),
			255,
			0,
		)

		if trans != nil {
			transform.TransformEval(trans.(*object.Transform).Body, textTexture, env)
		} else {
			transform.TransformEval(
				transform.BuiltInsTransform("default", textTexture.TextTexture.Width, textTexture.TextTexture.Height),
				textTexture, env,
			)
		}

		if sty != nil {
			style.StyleEval(sty.(*object.Style).Body, textTexture, env)
		}

		textTextureList = append(textTextureList, textTexture)

		config.LayerMutex.Lock()
		config.LayerList.Layers[2].ChangeTexture(textTexture, indexLayer)
		config.ChangeScreenTextureIndex(textTexture, indexScreenTexture)
		config.LayerMutex.Unlock()

		time.Sleep(time.Millisecond * 10)
	}

	// for i := 0; i < len(textTextureList)-1; i++ {
	// textTextureList[i].DestroyTexture()
	// }
}

func isInTexture(texture *core.SDL_Texture, x, y int) bool {
	switch texture.TextureType {
	case core.TEXTTEXTURE:
		return x >= core.ResizeInt(config.Width, config.ChangeWidth, texture.TextTexture.Xpos) &&
			x <= core.ResizeInt(config.Width, config.ChangeWidth, texture.TextTexture.Width)+core.ResizeInt(config.Width, config.ChangeWidth, texture.TextTexture.Xpos) &&
			y >= core.ResizeInt(config.Height, config.ChangeHeight, texture.TextTexture.Ypos) &&
			y <= core.ResizeInt(config.Height, config.ChangeHeight, texture.TextTexture.Height)+core.ResizeInt(config.Height, config.ChangeHeight, texture.TextTexture.Ypos)
	case core.IMAGETEXTURE:
		return x >= core.ResizeInt(config.Width, config.ChangeWidth, texture.ImageTexture.Xpos) &&
			x <= core.ResizeInt(config.Width, config.ChangeWidth, texture.ImageTexture.Width)+core.ResizeInt(config.Width, config.ChangeWidth, texture.ImageTexture.Xpos) &&
			y >= core.ResizeInt(config.Height, config.ChangeHeight, texture.ImageTexture.Ypos) &&
			y <= core.ResizeInt(config.Height, config.ChangeHeight, texture.ImageTexture.Height)+core.ResizeInt(config.Height, config.ChangeHeight, texture.ImageTexture.Ypos)
	}
	return false
}

func isFirstPriority(name string) bool {
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

func isScreenEnd(name string) bool {
	_, ok := config.ScreenAllIndex[name]
	return !ok
}

func isKeyWant(keyName string, inputKey uint8) bool {
	switch strings.ToLower(keyName) {
	case "0":
		return inputKey == uint8(core.SDLK_0)
	case "1":
		return inputKey == uint8(core.SDLK_1)
	case "2":
		return inputKey == uint8(core.SDLK_2)
	case "3":
		return inputKey == uint8(core.SDLK_3)
	case "4":
		return inputKey == uint8(core.SDLK_4)
	case "5":
		return inputKey == uint8(core.SDLK_5)
	case "6":
		return inputKey == uint8(core.SDLK_6)
	case "7":
		return inputKey == uint8(core.SDLK_7)
	case "8":
		return inputKey == uint8(core.SDLK_8)
	case "9":
		return inputKey == uint8(core.SDLK_9)
	case "a":
		return inputKey == uint8(core.SDLK_a)
	case "b":
		return inputKey == uint8(core.SDLK_b)
	case "c":
		return inputKey == uint8(core.SDLK_c)
	case "d":
		return inputKey == uint8(core.SDLK_d)
	case "e":
		return inputKey == uint8(core.SDLK_e)
	case "f":
		return inputKey == uint8(core.SDLK_f)
	case "g":
		return inputKey == uint8(core.SDLK_g)
	case "h":
		return inputKey == uint8(core.SDLK_h)
	case "i":
		return inputKey == uint8(core.SDLK_i)
	case "j":
		return inputKey == uint8(core.SDLK_j)
	case "k":
		return inputKey == uint8(core.SDLK_k)
	case "l":
		return inputKey == uint8(core.SDLK_l)
	case "m":
		return inputKey == uint8(core.SDLK_m)
	case "n":
		return inputKey == uint8(core.SDLK_n)
	case "o":
		return inputKey == uint8(core.SDLK_o)
	case "p":
		return inputKey == uint8(core.SDLK_p)
	case "q":
		return inputKey == uint8(core.SDLK_q)
	case "r":
		return inputKey == uint8(core.SDLK_r)
	case "s":
		return inputKey == uint8(core.SDLK_s)
	case "t":
		return inputKey == uint8(core.SDLK_t)
	case "u":
		return inputKey == uint8(core.SDLK_u)
	case "v":
		return inputKey == uint8(core.SDLK_v)
	case "w":
		return inputKey == uint8(core.SDLK_w)
	case "y":
		return inputKey == uint8(core.SDLK_y)
	case "z":
		return inputKey == uint8(core.SDLK_z)
	case "f1":
		return uint32(inputKey) == uint32(core.SDLK_F1)
	case "f2":
		return uint32(inputKey) == uint32(core.SDLK_F2)
	case "f3":
		return uint32(inputKey) == uint32(core.SDLK_F3)
	case "f4":
		return uint32(inputKey) == uint32(core.SDLK_F4)
	case "f5":
		return uint32(inputKey) == uint32(core.SDLK_F5)
	case "f6":
		return uint32(inputKey) == uint32(core.SDLK_F6)
	case "f7":
		return uint32(inputKey) == uint32(core.SDLK_F7)
	case "f8":
		return uint32(inputKey) == uint32(core.SDLK_F8)
	case "f9":
		return uint32(inputKey) == uint32(core.SDLK_F9)
	case "f10":
		return uint32(inputKey) == uint32(core.SDLK_F10)
	case "f11":
		return uint32(inputKey) == uint32(core.SDLK_F11)
	case "f12":
		return uint32(inputKey) == uint32(core.SDLK_F12)
	case "f13":
		return uint32(inputKey) == uint32(core.SDLK_F13)
	case "f14":
		return uint32(inputKey) == uint32(core.SDLK_F14)
	case "f15":
		return uint32(inputKey) == uint32(core.SDLK_F15)
	case "f16":
		return uint32(inputKey) == uint32(core.SDLK_F16)
	case "f17":
		return uint32(inputKey) == uint32(core.SDLK_F17)
	case "f18":
		return uint32(inputKey) == uint32(core.SDLK_F18)
	case "f19":
		return uint32(inputKey) == uint32(core.SDLK_F19)
	case "f20":
		return uint32(inputKey) == uint32(core.SDLK_F20)
	case "f21":
		return uint32(inputKey) == uint32(core.SDLK_F21)
	case "f22":
		return uint32(inputKey) == uint32(core.SDLK_F22)
	case "f23":
		return uint32(inputKey) == uint32(core.SDLK_F23)
	case "f24":
		return uint32(inputKey) == uint32(core.SDLK_F24)
	default:
		return false
	}
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
