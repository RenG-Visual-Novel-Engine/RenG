package evaluator

import (
	"RenG/interpreter/ast"
	"RenG/interpreter/object"
	"fmt"
)

var (
	index    = 0
	expIndex = 0
)

func evalStringLiteral(str *ast.StringLiteral, env *object.Environment) *object.String {
	result := &object.String{Value: str.Value}

	// TODO : 최적화하기
	// STRING + EXPRESSION + STRING 형식이 아닐시에 일어나는 오류
	// 강제적으로 [] [] 사이에는 띄어쓰기가 필요함 -> 언젠가 아이디어가 생기면 고쳐보자 -> 일단 고쳤지만 여러 최적화가 필요할듯

	for stringIndex := 0; stringIndex < len(str.Values); stringIndex++ {

		for isCurrentExp(index, str) {

			val := Eval(str.Exp[expIndex].Exp, env)

			switch value := val.(type) {
			case *object.Integer:
				s := fmt.Sprintf("%d", value.Value)
				result.Value += s
			case *object.Float:
				s := fmt.Sprintf("%f", value.Value)
				result.Value += s
			case *object.Boolean:
				s := fmt.Sprintf("%t", value.Value)
				result.Value += s
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
