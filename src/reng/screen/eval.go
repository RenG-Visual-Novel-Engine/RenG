package screen

import (
	"RenG/src/config"
	"RenG/src/core"
	"RenG/src/lang/ast"
	"RenG/src/lang/evaluator"
	"RenG/src/lang/object"
	"RenG/src/reng/transform"
	"fmt"
	"strconv"
)

func ScreenEval(node ast.Node, env *object.Environment, name string) object.Object {
	switch node := node.(type) {
	case *ast.BlockStatement:
		return evalBlockStatements(node, env, name)
	case *ast.ExpressionStatement:
		return ScreenEval(node.Expression, env, name)
	case *ast.PrefixExpression:
		if rightValue, ok := node.Right.(*ast.Identifier); ok {
			return evalAssignPrefixExpression(node.Operator, rightValue, env)
		} else {
			right := ScreenEval(node.Right, env, name)
			if isError(right) {
				return right
			}
			return evalPrefixExpression(node.Operator, right)
		}
	case *ast.InfixExpression:
		if leftValue, ok := node.Left.(*ast.Identifier); ok && isAssign(node.Operator) {
			right := ScreenEval(node.Right, env, name)
			if isError(right) {
				return right
			}

			return evalAssignInfixExpression(node.Operator, leftValue, right, env)
		} else {
			left := ScreenEval(node.Left, env, name)
			if isError(left) {
				return left
			}

			right := ScreenEval(node.Right, env, name)
			if isError(right) {
				return right
			}

			return evalInfixExpression(node.Operator, left, right)
		}
	case *ast.IfExpression:
		return evalIfExpression(node, env, name)
	case *ast.ForExpression:
		return evalForExpression(node, env, name)
	case *ast.WhileExpression:
		return evalWhileExpression(node, env, name)
	case *ast.CallFunctionExpression:
		function := ScreenEval(node.Function, env, name)
		if isError(function) {
			return function
		}

		args := evalExpressions(node.Arguments, env, name)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}

		return applyFunction(function, args, name)
	case *ast.IndexExpression:
		left := ScreenEval(node.Left, env, name)
		if isError(left) {
			return left
		}

		index := ScreenEval(node.Index, env, name)
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
		return evalStringLiteral(node, env, name)
	case *ast.ArrayLiteral:
		elements := evalExpressions(node.Elements, env, name)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.Array{Elements: elements}
	case *ast.ShowExpression:
		return evalShowExpression(node, env, name)
	case *ast.HideExpression:
		return evalHideExpression(node, env)
	case *ast.ImagebuttonExpression:
		return evalImagebuttonExpression(node, env, name)
	case *ast.TextbuttonExpression:
		return evalTextbuttonExpression(node, env, name)
	case *ast.WhoExpression:
		if config.Who != "" {
			return &object.String{Value: config.Who}
		}

		return newError("Not Init who value")
	case *ast.WhatExpression:
		if config.What != "" {
			return &object.String{Value: config.What}
		}

		return newError("Not Init what value")
	}
	return nil
}

func evalShowExpression(se *ast.ShowExpression, env *object.Environment, name string) object.Object {
	if texture, ok := config.TextureList.Get(se.Name.Value); ok {
		if trans, ok := env.Get(se.Transform.Value); ok {
			transform.TransformEval(trans.(*object.Transform).Body, texture, env)
		} else {
			transform.TransformEval(transform.TransformBuiltins["default"], texture, env)
		}

		config.ScreenAllIndex[name] = config.Screen{
			First: config.ScreenAllIndex[name].First,
			Count: config.ScreenAllIndex[name].Count + 1,
		}
		config.AddScreenTextureIndex(texture)

		config.LayerMutex.Lock()
		config.LayerList.Layers[2].AddNewTexture(texture)
		config.LayerMutex.Unlock()

		return nil
	} else if screens, ok := env.Get(se.Name.Value); ok {
		if screenObj, ok := screens.(*object.Screen); ok {
			if IsScreenEnd(screenObj.Name.Value) {
				ScreenMutex.Lock()
				config.ScreenAllIndex[screenObj.Name.Value] = config.Screen{First: config.ScreenIndex, Count: 0}
				config.ScreenPriority = append(config.ScreenPriority, screenObj.Name.Value)
				ScreenMutex.Unlock()
				return ScreenEval(screenObj.Body, env, screenObj.Name.Value)
			}
		}
	}

	/* else if video, ok := config.VideoList.Get(se.Name.Value); ok {
		if trans, ok := env.Get(se.Transform.Value); ok {
			go transform.TransformEval(trans.(*object.Transform).Body, video.Texture, env)
		} else {
			go transform.TransformEval(transform.TransformBuiltins["default"], video.Texture, env)
		}

		Set(name, config.ScreenIndex)
		config.AddScreenTextureIndex(video.Texture)

		// TODO
		go core.PlayVideo(video.Video, video.Texture, config.LayerMutex, config.LayerList, config.Renderer)
	}
	*/

	return nil
}

func evalHideExpression(he *ast.HideExpression, env *object.Environment) object.Object {
	if screens, ok := env.Get(he.Name.Value); ok {
		if screenObj, ok := screens.(*object.Screen); ok {
			ScreenMutex.Lock()
			config.DeleteScreen(screenObj.Name.Value)
			if index := FindScreenPriority(screenObj.Name.Value); index != -1 {
				config.ScreenPriority = append(config.ScreenPriority[:index], config.ScreenPriority[index+1:]...)
			}
			ScreenMutex.Unlock()
		}
	}
	return nil
}

func evalImagebuttonExpression(ie *ast.ImagebuttonExpression, env *object.Environment, name string) object.Object {
	if texture, ok := config.TextureList.Get(ie.MainImage.Value); ok {
		if trans, ok := env.Get(ie.Transform.Value); ok {
			transform.TransformEval(trans.(*object.Transform).Body, texture, env)
		} else {
			transform.TransformEval(transform.TransformBuiltins["default"], texture, env)
		}

		config.ScreenAllIndex[name] = config.Screen{
			First: config.ScreenAllIndex[name].First,
			Count: config.ScreenAllIndex[name].Count + 1,
		}
		config.AddScreenTextureIndex(texture)

		config.LayerMutex.Lock()
		config.LayerList.Layers[2].AddNewTexture(texture)
		config.LayerMutex.Unlock()

		go func() {
			for {
				event := <-config.MouseDownEventChan
				if IsScreenEnd(name) {
					return
				}
				if IsInTexture(texture, event.Mouse.Down.X, event.Mouse.Down.Y) && IsFirstPriority(name) {
					ScreenEval(ie.Action, env, name)
				}
				if IsScreenEnd(name) {
					return
				}
			}
		}()
	}
	return nil
}

func evalTextbuttonExpression(te *ast.TextbuttonExpression, env *object.Environment, name string) object.Object {
	text := ScreenEval(te.Text, env, name)
	if isError(text) {
		return text
	}

	if textObj, ok := text.(*object.String); ok {

		textTexture := config.MainFont.LoadFromRenderedText(
			textObj.Value,
			config.Renderer,
			config.Width, config.Height,
			core.CreateColor(235, 235, 235),
		)

		if trans, ok := env.Get(te.Transform.Value); ok {
			transform.TransformEval(trans.(*object.Transform).Body, textTexture, env)
		} else {
			transform.TransformEval(transform.TransformBuiltins["default"], textTexture, env)
		}

		config.ScreenAllIndex[name] = config.Screen{
			First: config.ScreenAllIndex[name].First,
			Count: config.ScreenAllIndex[name].Count + 1,
		}
		config.AddScreenTextureIndex(textTexture)

		config.LayerMutex.Lock()
		config.LayerList.Layers[2].AddNewTexture(textTexture)
		config.LayerMutex.Unlock()

		go func() {
			for {
				event := <-config.MouseDownEventChan
				if IsScreenEnd(name) {
					return
				}
				if IsInTexture(textTexture, event.Mouse.Down.X, event.Mouse.Down.Y) && IsFirstPriority(name) {
					ScreenEval(te.Action, env, name)
				}
				if IsScreenEnd(name) {
					return
				}
			}
		}()
	}

	return nil
}

func evalBlockStatements(block *ast.BlockStatement, env *object.Environment, name string) object.Object {
	var result object.Object

	for _, statement := range block.Statements {
		result = ScreenEval(statement, env, name)
		if result != nil {
			rt := result.Type()
			if rt == object.ERROR_OBJ {
				config.LayerMutex.Lock()
				config.LayerList.Layers[1].DeleteAllTexture()
				config.LayerList.Layers[0].AddNewTexture(
					config.MainFont.LoadFromRenderedText(
						result.(*object.Error).Message,
						config.Renderer,
						config.Width, config.Height,
						core.CreateColor(0, 0, 0),
					),
				)
				config.LayerMutex.Unlock()
				return result
			}
		}
	}

	return result
}

func evalExpressions(exps []ast.Expression, env *object.Environment, name string) []object.Object {
	var result []object.Object

	for _, e := range exps {
		evaluated := ScreenEval(e, env, name)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}

func evalStringLiteral(str *ast.StringLiteral, env *object.Environment, name string) *object.String {
	result := &object.String{Value: str.Value}

	// TODO : 최적화하기
	// 일단 고쳤지만 여러 최적화가 필요할듯
	var (
		index    = 0
		expIndex = 0
	)

	for stringIndex := 0; stringIndex < len(str.Values); stringIndex++ {

		for isCurrentExp(index, str) {

			val := ScreenEval(str.Exp[expIndex].Exp, env, name)

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

func evalIfExpression(ie *ast.IfExpression, env *object.Environment, name string) object.Object {
	condition := ScreenEval(ie.Condition, env, name)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return ScreenEval(ie.Consequence, env, name)
	}

	for _, ee := range ie.Elif {
		if ee != nil {
			elifCondition := ScreenEval(ee.Condition, env, name)
			if isError(elifCondition) {
				return elifCondition
			}
			if isTruthy(elifCondition) {
				return ScreenEval(ee.Consequence, env, name)
			}
		}
	}

	if ie.Alternative != nil {
		return ScreenEval(ie.Alternative, env, name)
	} else {
		return NULL
	}
}

func evalForExpression(node *ast.ForExpression, env *object.Environment, name string) object.Object {
	var define, condition, result, run object.Object

	define = ScreenEval(node.Define, env, name)
	if isError(define) {
		return define
	}

	condition = ScreenEval(node.Condition, env, name)
	if isError(condition) {
		return condition
	}

	for isTruthy(condition) {
		result = ScreenEval(node.Body, env, name)
		if isError(result) {
			return result
		}

		run = ScreenEval(node.Run, env, name)
		if isError(run) {
			return run
		}

		condition = ScreenEval(node.Condition, env, name)
		if isError(condition) {
			return condition
		}
	}
	return nil
}

func evalWhileExpression(node *ast.WhileExpression, env *object.Environment, name string) object.Object {
	condition := ScreenEval(node.Condition, env, name)
	if isError(condition) {
		return condition
	}

	for isTruthy(condition) {
		result := ScreenEval(node.Body, env, name)
		if isError(result) {
			return result
		}

		condition = ScreenEval(node.Condition, env, name)
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
