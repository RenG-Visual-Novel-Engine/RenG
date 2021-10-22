package rengeval

import (
	sdl "RenG/src/SDL"
	"RenG/src/ast"
	"RenG/src/config"
	"RenG/src/evaluator"
	"RenG/src/object"
	"RenG/src/transform"
	"fmt"
	"strconv"
	"sync"
)

var (
	LayerMutex = &sync.RWMutex{}
	MainMutex  = &sync.Mutex{}
)

func RengEval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {
	case *ast.BlockStatement:
		return evalRengBlockStatements(node, env)
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

func evalRengBlockStatements(block *ast.BlockStatement, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range block.Statements {
		result = RengEval(statement, env)
		if result != nil {
			switch result.Type() {
			case object.ERROR_OBJ:
				LayerMutex.Lock()
				config.LayerList.Layers[1].DeleteAllTexture()
				config.LayerList.Layers[2].DeleteAllTexture()
				config.LayerList.Layers[0].AddNewTexture(config.MainFont.LoadFromRenderedText(result.(*object.Error).Message, config.Renderer, sdl.CreateColor(0xFF, 0xFF, 0xFF)))
				LayerMutex.Unlock()
				return result
			case object.JUMP_LABEL_OBJ:
				return result
			}
		}
	}

	return result
}

func evalCallLabelExpression(cle *ast.CallLabelExpression, env *object.Environment) object.Object {
	if label, ok := env.Get(cle.Label.Value); ok {
		labelBody := label.(*object.Label).Body
		return RengEval(labelBody, env)
	} else {
		return newError("defined label %s", cle.Label.Value)
	}
}

func evalJumpLabelExpression(jle *ast.JumpLabelExpression) object.Object {
	return &object.JumpLabel{Label: jle.Label}
}

func evalShowExpression(se *ast.ShowExpression, env *object.Environment) object.Object {
	if texture, ok := config.TextureList.Get(se.Name.Value); ok {
		if trans, ok := env.Get(se.Transform.Value); ok {
			go transform.TransformEval(trans.(*object.Transform).Body, texture, env)
		} else {
			go transform.TransformEval(screenBuiltins["default"], texture, env)
		}

		addShowTextureIndex(texture)

		LayerMutex.Lock()
		config.LayerList.Layers[1].AddNewTexture(texture)
		LayerMutex.Unlock()

		return nil
	} else if video, ok := config.VideoList.Get(se.Name.Value); ok {
		if trans, ok := env.Get(se.Transform.Value); ok {
			go transform.TransformEval(trans.(*object.Transform).Body, video.Texture, env)
		} else {
			go transform.TransformEval(screenBuiltins["default"], video.Texture, env)
		}

		addShowTextureIndex(video.Texture)

		// TODO
		go playVideo(video)
	}

	return nil
}

func evalHideExpression(he *ast.HideExpression, env *object.Environment) object.Object {
	if texture, ok := config.TextureList.Get(he.Name.Value); ok {
		index := textureHasIndex(texture)
		LayerMutex.Lock()
		config.LayerList.Layers[1].DeleteTexture(index)
		LayerMutex.Unlock()
		updateShowTextureIndex(index)
		config.ShowIndex--
	}
	return nil
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

func evalPlayExpression(pe *ast.PlayExpression, env *object.Environment) object.Object {
	musicRootObject := RengEval(pe.Music, env)

	if musicRoot, ok := musicRootObject.(*object.String); ok {
		switch pe.Channel.Value {
		case "music":
			switch pe.Loop.Value {
			case "loop":
				go playMusic(config.Path+musicRoot.Value, true)
			case "noloop":
				go playMusic(config.Path+musicRoot.Value, false)
			default:
				return newError("It is not Loop or NoLoop. got=%s", pe.Loop.Value)
			}
		case "sound":
			go play(config.Path+musicRoot.Value, 0)
		case "voice":
			go play(config.Path+musicRoot.Value, 1)
		default:
			_, ok := config.ChannelList.GetChannel(pe.Channel.Value)
			if !ok {
				return newError("%s is not audio channel", pe.Channel.Value)
			}
		}
	} else {
		return newError("MusicRoot is not String")
	}

	return nil
}

func evalStopExpression(se *ast.StopExpression) object.Object {
	switch se.Channel.Value {
	case "music":
		sdl.StopMusic(-1)
	}
	return nil
}
