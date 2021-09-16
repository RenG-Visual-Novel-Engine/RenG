package evaluator

import (
	"RenG/interpreter/ast"
	"RenG/interpreter/object"
	"fmt"
)

func evalStringLiteral(str *ast.StringLiteral, env *object.Environment) *object.String {
	result := &object.String{Value: str.Value}

	// STRING + EXPRESSION + STRING 형식이 아닐시에 일어나는 오류
	// 강제적으로 [] [] 사이에는 띄어쓰기가 필요함 -> 언젠가 아이디어가 생기면 고쳐보자

	for i := 0; i < len(str.Exp); i++ {

		val := Eval(str.Exp[i], env)

		// 타입 검사
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

		result.Value += str.Values[i]
	}

	return result
}
