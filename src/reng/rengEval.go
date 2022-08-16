package reng

import (
	"RenG/src/config"
	"RenG/src/core"
	"RenG/src/lang/ast"
	"RenG/src/lang/evaluator"
	"RenG/src/lang/object"
	"RenG/src/reng/screen"
	"RenG/src/reng/transform"
	"fmt"
	"strconv"
)

func RengEval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {
	case *ast.BlockStatement:
		return evalRengBlockStatements(node, env)
	case *ast.ReturnStatement:
		val := RengEval(node.ReturnValue, env)
		if isError(val) {
			return val
		}

		return &object.ReturnValue{Value: val}
	case *ast.ExpressionStatement:
		return RengEval(node.Expression, env)
	case *ast.PrefixExpression:
		if rightValue, ok := node.Right.(*ast.Identifier); ok {
			return evalAssignPrefixExpression(node.Operator, rightValue, env)
		} else {
			right := RengEval(node.Right, env)
			if isError(right) {
				return right
			}
			return evalPrefixExpression(node.Operator, right)
		}
	case *ast.InfixExpression:
		if leftValue, ok := node.Left.(*ast.Identifier); ok && isAssign(node.Operator) {
			right := RengEval(node.Right, env)
			if isError(right) {
				return right
			}

			return evalAssignInfixExpression(node.Operator, leftValue, right, env)
		} else {
			left := RengEval(node.Left, env)
			if isError(left) {
				return left
			}

			right := RengEval(node.Right, env)
			if isError(right) {
				return right
			}

			return evalInfixExpression(node.Operator, left, right)
		}
	case *ast.IfExpression:
		return evalIfExpression(node, env)
	case *ast.ForExpression:
		return evalForExpression(node, env)
	case *ast.WhileExpression:
		return evalWhileExpression(node, env)
	case *ast.CallFunctionExpression:
		function := RengEval(node.Function, env)
		if isError(function) {
			return function
		}

		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}

		return applyFunction(function, args)
	case *ast.IndexExpression:
		left := RengEval(node.Left, env)
		if isError(left) {
			return left
		}

		index := RengEval(node.Index, env)
		if isError(index) {
			return index
		}

		return evalIndexExpression(left, index)
	case *ast.Identifier:
		return evalIdentifier(node, env)
	case *ast.Boolean:
		return &object.Boolean{Value: node.Value}
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}
	case *ast.FloatLiteral:
		return &object.Float{Value: node.Value}
	case *ast.StringLiteral:
		return evalStringLiteral(node, env)
	case *ast.ArrayLiteral:
		elements := evalExpressions(node.Elements, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.Array{Elements: elements}
	case *ast.CallLabelExpression:
		return evalCallLabelExpression(node, env)
	case *ast.JumpLabelExpression:
		return evalJumpLabelExpression(node)
	case *ast.MenuExpression:
		return evalMenuExpression(node, env)
	case *ast.ShowExpression:
		return evalShowExpression(node, env)
	case *ast.HideExpression:
		return evalHideExpression(node, env)
	case *ast.PlayExpression:
		return evalPlayExpression(node, env)
	case *ast.StopExpression:
		return evalStopExpression(node)
	}
	return nil
}

// TODO : Entry Point 재설정
func evalRengBlockStatements(block *ast.BlockStatement, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range block.Statements {
		result = RengEval(statement, env)
		if result != nil {
			switch result.Type() {
			case object.RETURN_VALUE_OBJ:
				return result
			case object.ERROR_OBJ:
				config.LayerMutex.Lock()
				config.LayerList.Layers[1].DeleteAllTexture()
				config.LayerList.Layers[2].DeleteAllTexture()
				config.LayerList.Layers[0].AddNewTexture(
					config.MainFont.LoadFromRenderedText(
						result.(*object.Error).Message,
						config.Renderer,
						uint(config.Width),
						core.CreateColor(0xFF, 0xFF, 0xFF),
						255,
						0,
					),
				)
				config.LayerMutex.Unlock()
				return result
			case object.JUMP_LABEL_OBJ:
				return result
			case object.STRING_OBJ:
				say, ok := env.Get("say")
				if !ok {
					return newError("Not Defined Say Screen")
				}

				config.What = result.(*object.String).Value

				screen.ScreenEval(say.(*object.Screen).Body, env, "say")

				// TODO : 왼쪽 마우스 클릭만 인식하도록 변경해야 함.
				<-config.MouseDownEventChan

				config.DeleteScreen("say")

				config.Who = ""
				config.What = ""
			case object.CHARACTER_OBJ:
				config.Who = result.(*object.Character).Name.Value
				config.WhoColor = result.(*object.Character).Color
			}
		}
	}
	if result.Type() != object.STRING_OBJ {
		return result
	}
	return NULL
}

func evalCallLabelExpression(cle *ast.CallLabelExpression, env *object.Environment) object.Object {
	if label, ok := env.Get(cle.Label.Value); ok {
		RengEval(label.(*object.Label).Body, env)
	} else {
		return newError("defined label %s", cle.Label.Value)
	}
	return NULL
}

func evalJumpLabelExpression(jle *ast.JumpLabelExpression) object.Object {
	return &object.JumpLabel{Label: jle.Label}
}

func evalMenuExpression(me *ast.MenuExpression, env *object.Environment) object.Object {
	var (
		key  object.Object
		keys []object.Object
	)
	config.IsNowMenu = true

	for i := 0; i < len(me.Key); i++ {
		key = RengEval(me.Key[i], env)
		if isError(key) {
			return key
		}

		if _, ok := key.(*object.String); !ok {
			return newError("menu key %v is not string value", key)
		}

		keys = append(keys, key)
	}

	config.Items = &object.Array{
		Elements: keys,
	}

	choice, ok := env.Get("choice")
	if !ok {
		return newError("chocie screen is not defined")
	}

	if choiceScreen, ok := choice.(*object.Screen); ok {
		screen.ScreenMutex.Lock()
		config.ScreenAllIndex[choiceScreen.Name.Value] = config.Screen{First: config.ScreenIndex, Count: 0}
		config.ScreenPriority = append(config.ScreenPriority, choiceScreen.Name.Value)
		screen.ScreenMutex.Unlock()

		screen.ScreenEval(choiceScreen.Body, env, choiceScreen.Name.Value)
	} else {
		return newError("choice ident is not screen object")
	}

	event := <-config.SelectMenuIndex

	config.IsNowMenu = false
	config.IsNowMenuIndex = 0

	screen.ScreenMutex.Lock()
	config.DeleteScreen("choice")
	if index := screen.FindScreenPriority("choice"); index != -1 {
		config.ScreenPriority = append(config.ScreenPriority[:index], config.ScreenPriority[index+1:]...)
	}
	screen.ScreenMutex.Unlock()

	return RengEval(me.Action[event], env)
}

func evalShowExpression(se *ast.ShowExpression, env *object.Environment) object.Object {
	if texture, ok := config.TextureList.Get(se.Name.Value); ok {
		if trans, ok := env.Get(se.Transform.Value); ok {
			transform.TransformEval(trans.(*object.Transform).Body, texture, env)
		} else {
			transform.TransformEval(
				transform.BuiltInsTransform("default", texture.ImageTexture.Width, texture.ImageTexture.Height),
				texture, env,
			)
		}

		config.AddShowTextureIndex(texture)

		config.LayerMutex.Lock()
		config.LayerList.Layers[1].AddNewTexture(texture)
		config.LayerMutex.Unlock()

		return nil
	} else if screens, ok := env.Get(se.Name.Value); ok {
		if screenObj, ok := screens.(*object.Screen); ok {
			screen.ScreenMutex.Lock()
			config.ScreenAllIndex[screenObj.Name.Value] = config.Screen{First: config.ScreenIndex, Count: 0}
			config.ScreenPriority = append(config.ScreenPriority, screenObj.Name.Value)
			screen.ScreenMutex.Unlock()
			return screen.ScreenEval(screenObj.Body, env, screenObj.Name.Value)
		}
	}

	/* else if video, ok := config.VideoList.Get(se.Name.Value); ok {
		if trans, ok := env.Get(se.Transform.Value); ok {
			transform.TransformEval(trans.(*object.Transform).Body, video.Texture, env)
		} else {
			// transform.TransformEval(transform.BuiltInsTransform("default", video.), video.Texture, env)
		}

		config.AddShowTextureIndex(video.Texture)

		// TODO
		go core.PlayVideo(video.Video, video.Texture, config.LayerMutex, config.LayerList, config.Renderer)
	}
	*/

	return NULL
}

func evalHideExpression(he *ast.HideExpression, env *object.Environment) object.Object {
	if texture, ok := config.TextureList.Get(he.Name.Value); ok {
		index := config.ShowTextureHasIndex(texture)

		config.LayerMutex.Lock()
		config.LayerList.Layers[1].DeleteTexture(index)
		config.LayerMutex.Unlock()

		config.DeleteShowTextureIndex(index)
		config.ShowIndex--
	} else if screens, ok := env.Get(he.Name.Value); ok {
		if screenObj, ok := screens.(*object.Screen); ok {
			screen.ScreenMutex.Lock()
			config.DeleteScreen(screenObj.Name.Value)
			if index := screen.FindScreenPriority(screenObj.Name.Value); index != -1 {
				config.ScreenPriority = append(config.ScreenPriority[:index], config.ScreenPriority[index+1:]...)
			}
			screen.ScreenMutex.Unlock()
		}
	}
	return NULL
}

func evalPlayExpression(pe *ast.PlayExpression, env *object.Environment) object.Object {
	musicRootObject := RengEval(pe.Music, env)

	if musicRoot, ok := musicRootObject.(*object.String); ok {
		switch pe.Channel.Value {
		case "music":
			switch pe.Loop.Value {
			case "loop":
				go playBGMusic(config.Path+musicRoot.Value, true)
			case "noloop":
				go playBGMusic(config.Path+musicRoot.Value, false)
			default:
				return newError("It is not loop or noloop. got=%s", pe.Loop.Value)
			}
		case "sound":
			go playSound(config.Path+musicRoot.Value, 0)
		case "voice":
			go playSound(config.Path+musicRoot.Value, 1)
		default:
			// TODO : 사용자 정의 채널 미구현
			_, ok := config.ChannelList.GetChannel(pe.Channel.Value)
			if !ok {
				return newError("%s is not audio channel", pe.Channel.Value)
			}
		}
	} else {
		return newError("MusicRoot is not String")
	}

	return NULL
}

func evalStopExpression(se *ast.StopExpression) object.Object {
	switch se.Channel.Value {
	case "music":
		core.StopMusic(-1)
	case "sound":
		core.StopMusic(0)
	case "voice":
		core.StopMusic(1)
	}
	return NULL
}

func evalExpressions(exps []ast.Expression, env *object.Environment) []object.Object {
	var result []object.Object

	for _, e := range exps {
		evaluated := RengEval(e, env)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}

func evalStringLiteral(str *ast.StringLiteral, env *object.Environment) *object.String {
	result := &object.String{Value: str.Value}

	// TODO : 최적화하기
	// 일단 고쳤지만 여러 최적화가 필요할듯
	var (
		index    = 0
		expIndex = 0
	)

	for stringIndex := 0; stringIndex < len(str.Values); stringIndex++ {

		for isCurrentExp(index, str) {

			val := RengEval(str.Exp[expIndex].Exp, env)

			switch value := val.(type) {
			case *object.Integer:
				result.Value += strconv.Itoa(int(value.Value))
			case *object.Float:
				result.Value += fmt.Sprintf("%f", value.Value)
			case *object.Boolean:
				result.Value += strconv.FormatBool(value.Value)
			case *object.String:
				result.Value += value.Value
			default:
				result.Value = "ErrorType"
			}

			expIndex++
			index++
		}

		result.Value += str.Values[stringIndex].Str

		index++
	}

	return result
}

func evalIndexExpression(left, index object.Object) object.Object {
	switch {
	case left.Type() == object.ARRAY_OBJ && index.Type() == object.INTEGER_OBJ:
		return evalArrayIndexExpression(left, index)
	case left.Type() == object.STRING_OBJ && index.Type() == object.INTEGER_OBJ:
		str := left.(*object.String).Value
		idx := index.(*object.Integer).Value
		max := int64(len(str) - 1)
		if idx < 0 || idx > max {
			return NULL
		}
		return &object.String{Value: string(str[idx])}
	default:
		return newError("index operator not supported : %s", left.Type())
	}
}

func evalArrayIndexExpression(array, index object.Object) object.Object {
	arrayObject := array.(*object.Array)
	idx := index.(*object.Integer).Value
	max := int64(len(arrayObject.Elements) - 1)
	if idx < 0 || idx > max {
		return NULL
	}
	return arrayObject.Elements[idx]
}

func evalIfExpression(ie *ast.IfExpression, env *object.Environment) object.Object {
	condition := RengEval(ie.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return RengEval(ie.Consequence, env)
	}

	for _, ee := range ie.Elif {
		if ee != nil {
			elifCondition := RengEval(ee.Condition, env)
			if isError(elifCondition) {
				return elifCondition
			}
			if isTruthy(elifCondition) {
				return RengEval(ee.Consequence, env)
			}
		}
	}

	if ie.Alternative != nil {
		return RengEval(ie.Alternative, env)
	} else {
		return NULL
	}
}

func evalForExpression(node *ast.ForExpression, env *object.Environment) object.Object {
	var define, condition, result, run object.Object

	define = RengEval(node.Define, env)
	if isError(define) {
		return define
	}

	condition = RengEval(node.Condition, env)
	if isError(condition) {
		return condition
	}

	for isTruthy(condition) {
		result = RengEval(node.Body, env)
		if isError(result) {
			return result
		}

		if _, ok := result.(*object.ReturnValue); ok {
			return result
		} else if _, ok = result.(*object.JumpLabel); ok {
			return result
		}

		run = RengEval(node.Run, env)
		if isError(run) {
			return run
		}

		condition = RengEval(node.Condition, env)
		if isError(condition) {
			return condition
		}
	}
	return nil
}

func evalWhileExpression(node *ast.WhileExpression, env *object.Environment) object.Object {
	condition := RengEval(node.Condition, env)
	if isError(condition) {
		return condition
	}

	for isTruthy(condition) {
		result := RengEval(node.Body, env)
		if isError(result) {
			return result
		}

		if _, ok := result.(*object.ReturnValue); ok {
			return result
		} else if _, ok = result.(*object.JumpLabel); ok {
			return result
		}

		condition = RengEval(node.Condition, env)
		if isError(condition) {
			return condition
		}
	}
	return nil
}

func evalIdentifier(node *ast.Identifier, env *object.Environment) object.Object {
	if val, ok := env.Get(node.Value); ok {
		return val
	}

	if builtin, ok := evaluator.FunctionBuiltins[node.Value]; ok {
		return builtin
	}

	return newError("identifier not found: " + node.Value)
}
