package style

import (
	"RenG/src/config"
	"RenG/src/core"
	"RenG/src/lang/ast"
	"RenG/src/lang/evaluator"
	"RenG/src/lang/object"
	"fmt"
	"strconv"
)

func StyleEval(node ast.Node, texture *core.SDL_Texture, env *object.Environment) object.Object {
	switch node := node.(type) {
	case *ast.BlockStatement:
		return evalBlockStatements(node, texture, env)
	case *ast.ExpressionStatement:
		return StyleEval(node.Expression, texture, env)
	case *ast.PrefixExpression:
		if rightValue, ok := node.Right.(*ast.Identifier); ok {
			return evalAssignPrefixExpression(node.Operator, rightValue, env)
		} else {
			right := StyleEval(node.Right, texture, env)
			if isError(right) {
				return right
			}
			return evalPrefixExpression(node.Operator, right)
		}
	case *ast.InfixExpression:
		if leftValue, ok := node.Left.(*ast.Identifier); ok && isAssign(node.Operator) {
			right := StyleEval(node.Right, texture, env)
			if isError(right) {
				return right
			}

			return evalAssignInfixExpression(node.Operator, leftValue, right, env)
		} else {
			left := StyleEval(node.Left, texture, env)
			if isError(left) {
				return left
			}

			right := StyleEval(node.Right, texture, env)
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
		function := StyleEval(node.Function, texture, env)
		if isError(function) {
			return function
		}

		args := evalExpressions(node.Arguments, texture, env)
		if len(args) == 1 && isError(args[0]) {
			return args[0]
		}

		return applyFunction(function, texture, args)
	case *ast.IndexExpression:
		left := StyleEval(node.Left, texture, env)
		if isError(left) {
			return left
		}

		index := StyleEval(node.Index, texture, env)
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
	case *ast.ColorExpression:
		colorObj := StyleEval(node.Value, texture, env)
		if isError(colorObj) {
			return colorObj
		}

		if color, ok := colorObj.(*object.String); ok {
			hex := make([]int64, 3)
			switch color.Value[:1] {
			case "#":
				hex[0], _ = strconv.ParseInt(color.Value[1:3], 16, 32)
				hex[1], _ = strconv.ParseInt(color.Value[3:5], 16, 32)
				hex[2], _ = strconv.ParseInt(color.Value[5:7], 16, 32)
			default:
				return newError("Color support only hex code")
			}

			config.MainFont.ChangeTextColor(texture, config.Renderer, texture.TextTexture.Text, core.CreateColor(int(hex[0]), int(hex[1]), int(hex[2])))
		}
	}
	return nil
}

func evalBlockStatements(block *ast.BlockStatement, texture *core.SDL_Texture, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range block.Statements {
		result = StyleEval(statement, texture, env)
		if result != nil {
			rt := result.Type()
			if rt == object.ERROR_OBJ {
				return result
			}
		}
	}

	return result
}

func evalExpressions(exps []ast.Expression, texture *core.SDL_Texture, env *object.Environment) []object.Object {
	var result []object.Object

	for _, e := range exps {
		evaluated := StyleEval(e, texture, env)
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

			val := StyleEval(str.Exp[expIndex].Exp, texture, env)

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
			return object.NULL
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
		return object.NULL
	}
	return arrayObject.Elements[idx]
}

func evalIfExpression(ie *ast.IfExpression, texture *core.SDL_Texture, env *object.Environment) object.Object {
	condition := StyleEval(ie.Condition, texture, env)
	if isError(condition) {
		return condition
	}

	if isTruthy(condition) {
		return StyleEval(ie.Consequence, texture, env)
	}

	for _, ee := range ie.Elif {
		if ee != nil {
			elifCondition := StyleEval(ee.Condition, texture, env)
			if isError(elifCondition) {
				return elifCondition
			}
			if isTruthy(elifCondition) {
				return StyleEval(ee.Consequence, texture, env)
			}
		}
	}

	if ie.Alternative != nil {
		return StyleEval(ie.Alternative, texture, env)
	} else {
		return object.NULL
	}
}

func evalForExpression(node *ast.ForExpression, texture *core.SDL_Texture, env *object.Environment) object.Object {
	var define, condition, result, run object.Object

	define = StyleEval(node.Define, texture, env)
	if isError(define) {
		return define
	}

	condition = StyleEval(node.Condition, texture, env)
	if isError(condition) {
		return condition
	}

	for isTruthy(condition) {
		result = StyleEval(node.Body, texture, env)
		if isError(result) {
			return result
		}

		run = StyleEval(node.Run, texture, env)
		if isError(run) {
			return run
		}

		condition = StyleEval(node.Condition, texture, env)
		if isError(condition) {
			return condition
		}
	}
	return nil
}

func evalWhileExpression(node *ast.WhileExpression, texture *core.SDL_Texture, env *object.Environment) object.Object {
	condition := StyleEval(node.Condition, texture, env)
	if isError(condition) {
		return condition
	}

	for isTruthy(condition) {
		result := StyleEval(node.Body, texture, env)
		if isError(result) {
			return result
		}

		condition = StyleEval(node.Condition, texture, env)
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
