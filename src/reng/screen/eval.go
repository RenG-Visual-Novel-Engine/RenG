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

func ScreenEval(node ast.Node, env *object.Environment) object.Object {
	switch node := node.(type) {
	case *ast.BlockStatement:
		return evalBlockStatements(node, env)
	case *ast.ExpressionStatement:
		return ScreenEval(node.Expression, env)
	case *ast.PrefixExpression:
		if rightValue, ok := node.Right.(*ast.Identifier); ok {
			return evalAssignPrefixExpression(node.Operator, rightValue, env)
		} else {
			right := ScreenEval(node.Right, env)
			if isError(right) {
				return right
			}
			return evalPrefixExpression(node.Operator, right)
		}
	case *ast.InfixExpression:
		if leftValue, ok := node.Left.(*ast.Identifier); ok && isAssign(node.Operator) {
			right := ScreenEval(node.Right, env)
			if isError(right) {
				return right
			}

			return evalAssignInfixExpression(node.Operator, leftValue, right, env)
		} else {
			left := ScreenEval(node.Left, env)
			if isError(left) {
				return left
			}

			right := ScreenEval(node.Right, env)
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
		function := ScreenEval(node.Function, env)
		if isError(function) {
			return function
		}

		args := evalExpressions(node.Arguments, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}

		return applyFunction(function, args)
	case *ast.IndexExpression:
		left := ScreenEval(node.Left, env)
		if isError(left) {
			return left
		}

		index := ScreenEval(node.Index, env)
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
	case *ast.ShowExpression:
		return evalShowExpression(node, env)
	}
	return nil
}

func evalShowExpression(se *ast.ShowExpression, env *object.Environment) object.Object {
	if texture, ok := config.TextureList.Get(se.Name.Value); ok {
		if trans, ok := env.Get(se.Transform.Value); ok {
			go transform.TransformEval(trans.(*object.Transform).Body, texture, env)
		} else {
			go transform.TransformEval(transform.TransformBuiltins["default"], texture, env)
		}

		addShowTextureIndex(texture)

		config.LayerMutex.Lock()
		config.LayerList.Layers[1].AddNewTexture(texture)
		config.LayerMutex.Unlock()

		return nil
	} else if video, ok := config.VideoList.Get(se.Name.Value); ok {
		if trans, ok := env.Get(se.Transform.Value); ok {
			go transform.TransformEval(trans.(*object.Transform).Body, video.Texture, env)
		} else {
			go transform.TransformEval(transform.TransformBuiltins["default"], video.Texture, env)
		}

		addShowTextureIndex(video.Texture)

		// TODO
		go core.PlayVideo(video.Video, video.Texture, config.LayerMutex, config.LayerList, config.Renderer)
	}

	return nil
}

func evalBlockStatements(block *ast.BlockStatement, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range block.Statements {
		result = ScreenEval(statement, env)
		if result != nil {
			rt := result.Type()
			if rt == object.ERROR_OBJ {
				return result
			}
		}
	}

	return result
}

func evalExpressions(exps []ast.Expression, env *object.Environment) []object.Object {
	var result []object.Object

	for _, e := range exps {
		evaluated := ScreenEval(e, env)
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

			val := ScreenEval(str.Exp[expIndex].Exp, env)

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
	condition := ScreenEval(ie.Condition, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return ScreenEval(ie.Consequence, env)
	}

	for _, ee := range ie.Elif {
		if ee != nil {
			elifCondition := ScreenEval(ee.Condition, env)
			if isError(elifCondition) {
				return elifCondition
			}
			if isTruthy(elifCondition) {
				return ScreenEval(ee.Consequence, env)
			}
		}
	}

	if ie.Alternative != nil {
		return ScreenEval(ie.Alternative, env)
	} else {
		return NULL
	}
}

func evalForExpression(node *ast.ForExpression, env *object.Environment) object.Object {
	var define, condition, result, run object.Object

	define = ScreenEval(node.Define, env)
	if isError(define) {
		return define
	}

	condition = ScreenEval(node.Condition, env)
	if isError(condition) {
		return condition
	}

	for isTruthy(condition) {
		result = ScreenEval(node.Body, env)
		if isError(result) {
			return result
		}

		run = ScreenEval(node.Run, env)
		if isError(run) {
			return run
		}

		condition = ScreenEval(node.Condition, env)
		if isError(condition) {
			return condition
		}
	}
	return nil
}

func evalWhileExpression(node *ast.WhileExpression, env *object.Environment) object.Object {
	condition := ScreenEval(node.Condition, env)
	if isError(condition) {
		return condition
	}

	for isTruthy(condition) {
		result := ScreenEval(node.Body, env)
		if isError(result) {
			return result
		}

		condition = ScreenEval(node.Condition, env)
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
