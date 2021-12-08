package transform

import (
	"RenG/src/core"
	"RenG/src/lang/ast"
	"RenG/src/lang/evaluator"
	"RenG/src/lang/object"
	"fmt"
	"strconv"
)

func TransformEval(node ast.Node, texture *core.SDL_Texture, env *object.Environment) object.Object {
	switch node := node.(type) {
	case *ast.BlockStatement:
		return evalBlockStatements(node, texture, env)
	case *ast.ExpressionStatement:
		return TransformEval(node.Expression, texture, env)
	case *ast.PrefixExpression:
		if rightValue, ok := node.Right.(*ast.Identifier); ok {
			return evalAssignPrefixExpression(node.Operator, rightValue, env)
		} else {
			right := TransformEval(node.Right, texture, env)
			if isError(right) {
				return right
			}
			return evalPrefixExpression(node.Operator, right)
		}
	case *ast.InfixExpression:
		if leftValue, ok := node.Left.(*ast.Identifier); ok && isAssign(node.Operator) {
			right := TransformEval(node.Right, texture, env)
			if isError(right) {
				return right
			}

			return evalAssignInfixExpression(node.Operator, leftValue, right, env)
		} else {
			left := TransformEval(node.Left, texture, env)
			if isError(left) {
				return left
			}

			right := TransformEval(node.Right, texture, env)
			if isError(right) {
				return right
			}

			return evalInfixExpression(node.Operator, left, right)
		}
	case *ast.IfExpression:
		return evalIfExpression(node, texture, env)
	case *ast.ForExpression:
		return evalForExpression(node, texture, env)
	case *ast.WhileExpression:
		return evalWhileExpression(node, texture, env)
	case *ast.CallFunctionExpression:
		function := TransformEval(node.Function, texture, env)
		if isError(function) {
			return function
		}

		args := evalExpressions(node.Arguments, texture, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}

		return applyFunction(function, texture, args)
	case *ast.IndexExpression:
		left := TransformEval(node.Left, texture, env)
		if isError(left) {
			return left
		}

		index := TransformEval(node.Index, texture, env)
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
		return evalStringLiteral(node, texture, env)
	case *ast.ArrayLiteral:
		elements := evalExpressions(node.Elements, texture, env)
		if len(elements) == 1 && isError(elements[0]) {
			return elements[0]
		}
		return &object.Array{Elements: elements}
	case *ast.TransformExpression:
		return evalTransformExpression(node, texture, env)
	case *ast.XPosExpression:
		result := TransformEval(node.Value, texture, env)
		switch xpos := result.(type) {
		case *object.Integer:
			switch texture.TextureType {
			case core.TEXTTEXTURE:
				texture.TextTexture.Xpos = int(xpos.Value)
			case core.IMAGETEXTURE:
				texture.ImageTexture.Xpos = int(xpos.Value)
			}
		case *object.Float:
			switch texture.TextureType {
			case core.TEXTTEXTURE:
				texture.TextTexture.Xpos = int(xpos.Value)
			case core.IMAGETEXTURE:
				texture.ImageTexture.Xpos = int(xpos.Value)
			}
		default:
			return newError("xpos expression isn't integer or float")
		}
	case *ast.YPosExpression:
		result := TransformEval(node.Value, texture, env)
		if isError(result) {
			return result
		}

		switch ypos := result.(type) {
		case *object.Integer:
			switch texture.TextureType {
			case core.TEXTTEXTURE:
				texture.TextTexture.Ypos = int(ypos.Value)
			case core.IMAGETEXTURE:
				texture.ImageTexture.Ypos = int(ypos.Value)
			}
		case *object.Float:
			switch texture.TextureType {
			case core.TEXTTEXTURE:
				texture.TextTexture.Ypos = int(ypos.Value)
			case core.IMAGETEXTURE:
				texture.ImageTexture.Ypos = int(ypos.Value)
			}
		default:
			return newError("ypos expression isn't integer or float")
		}
	case *ast.XSizeExpression:
		result := TransformEval(node.Value, texture, env)
		if isError(result) {
			return result
		}

		switch xsize := result.(type) {
		case *object.Integer:
			switch texture.TextureType {
			case core.TEXTTEXTURE:
				texture.TextTexture.Width = int(xsize.Value)
			case core.IMAGETEXTURE:
				texture.ImageTexture.Width = int(xsize.Value)
			}
		case *object.Float:
			switch texture.TextureType {
			case core.TEXTTEXTURE:
				texture.TextTexture.Width = int(xsize.Value)
			case core.IMAGETEXTURE:
				texture.ImageTexture.Width = int(xsize.Value)
			}
		default:
			return newError("xsize expression isn't integer or float")
		}
	case *ast.YSizeExpression:
		result := TransformEval(node.Value, texture, env)
		if isError(result) {
			return result
		}

		switch ysize := result.(type) {
		case *object.Integer:
			switch texture.TextureType {
			case core.TEXTTEXTURE:
				texture.TextTexture.Height = int(ysize.Value)
			case core.IMAGETEXTURE:
				texture.ImageTexture.Height = int(ysize.Value)
			}
		case *object.Float:
			switch texture.TextureType {
			case core.TEXTTEXTURE:
				texture.TextTexture.Height = int(ysize.Value)
			case core.IMAGETEXTURE:
				texture.ImageTexture.Height = int(ysize.Value)
			}
		default:
			return newError("ysize expression isn't integer or float")
		}
	case *ast.RotateExpression:
		result := TransformEval(node.Value, texture, env)
		if isError(result) {
			return result
		}

		switch rotate := result.(type) {
		case *object.Integer:
			switch texture.TextureType {
			case core.TEXTTEXTURE:
				texture.TextTexture.Degree = float64(rotate.Value)
			case core.IMAGETEXTURE:
				texture.ImageTexture.Degree = float64(rotate.Value)
			}
		case *object.Float:
			switch texture.TextureType {
			case core.TEXTTEXTURE:
				texture.TextTexture.Degree = rotate.Value
			case core.IMAGETEXTURE:
				texture.ImageTexture.Degree = rotate.Value
			}
		default:
			return newError("rotate expression isn't integer or float")
		}
	case *ast.AlphaExpression:
		result := TransformEval(node.Value, texture, env)
		if isError(result) {
			return result
		}

		switch alpha := result.(type) {
		case *object.Integer:
			switch texture.TextureType {
			case core.TEXTTEXTURE:
				texture.TextTexture.Alpha = uint8(alpha.Value)
			case core.IMAGETEXTURE:
				texture.ImageTexture.Alpha = uint8(alpha.Value)
			}
		case *object.Float:
			switch texture.TextureType {
			case core.TEXTTEXTURE:
				texture.TextTexture.Alpha = uint8(alpha.Value)
			case core.IMAGETEXTURE:
				texture.ImageTexture.Alpha = uint8(alpha.Value)
			}
		default:
			return newError("alpha expression isn't integer or float")
		}
	}
	return nil
}

func evalBlockStatements(block *ast.BlockStatement, texture *core.SDL_Texture, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range block.Statements {
		result = TransformEval(statement, texture, env)
		if result != nil {
			rt := result.Type()
			if rt == object.ERROR_OBJ {
				return result
			}
		}
	}

	return result
}

func evalTransformExpression(transform *ast.TransformExpression, texture *core.SDL_Texture, env *object.Environment) object.Object {
	TransformEval(transform.Body, texture, env)
	return NULL
}

func evalExpressions(exps []ast.Expression, texture *core.SDL_Texture, env *object.Environment) []object.Object {
	var result []object.Object

	for _, e := range exps {
		evaluated := TransformEval(e, texture, env)
		if isError(evaluated) {
			return []object.Object{evaluated}
		}
		result = append(result, evaluated)
	}

	return result
}

func evalStringLiteral(str *ast.StringLiteral, texture *core.SDL_Texture, env *object.Environment) *object.String {
	result := &object.String{Value: str.Value}

	// TODO : 최적화하기
	// 일단 고쳤지만 여러 최적화가 필요할듯
	var (
		index    = 0
		expIndex = 0
	)

	for stringIndex := 0; stringIndex < len(str.Values); stringIndex++ {

		for isCurrentExp(index, str) {

			val := TransformEval(str.Exp[expIndex].Exp, texture, env)

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

func evalIfExpression(ie *ast.IfExpression, texture *core.SDL_Texture, env *object.Environment) object.Object {
	condition := TransformEval(ie.Condition, texture, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return TransformEval(ie.Consequence, texture, env)
	}

	for _, ee := range ie.Elif {
		if ee != nil {
			elifCondition := TransformEval(ee.Condition, texture, env)
			if isError(elifCondition) {
				return elifCondition
			}
			if isTruthy(elifCondition) {
				return TransformEval(ee.Consequence, texture, env)
			}
		}
	}

	if ie.Alternative != nil {
		return TransformEval(ie.Alternative, texture, env)
	} else {
		return NULL
	}
}

func evalForExpression(node *ast.ForExpression, texture *core.SDL_Texture, env *object.Environment) object.Object {
	var define, condition, result, run object.Object

	define = TransformEval(node.Define, texture, env)
	if isError(define) {
		return define
	}

	condition = TransformEval(node.Condition, texture, env)
	if isError(condition) {
		return condition
	}

	for isTruthy(condition) {
		result = TransformEval(node.Body, texture, env)
		if isError(result) {
			return result
		}

		run = TransformEval(node.Run, texture, env)
		if isError(run) {
			return run
		}

		condition = TransformEval(node.Condition, texture, env)
		if isError(condition) {
			return condition
		}
	}
	return NULL
}

func evalWhileExpression(node *ast.WhileExpression, texture *core.SDL_Texture, env *object.Environment) object.Object {
	condition := TransformEval(node.Condition, texture, env)
	if isError(condition) {
		return condition
	}

	for isTruthy(condition) {
		result := TransformEval(node.Body, texture, env)
		if isError(result) {
			return result
		}

		condition = TransformEval(node.Condition, texture, env)
		if isError(condition) {
			return condition
		}
	}
	return NULL
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
