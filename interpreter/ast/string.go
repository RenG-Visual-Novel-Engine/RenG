package ast

import (
	"RenG/interpreter/token"
	"bytes"
)

//TODO : String 함수 완성

type StringLiteral struct {
	Token  token.Token
	Value  string
	Values []StringIndex
	Exp    []ExpressionIndex
}

type StringIndex struct {
	Str   string
	Index int
}

type ExpressionIndex struct {
	Exp   Expression
	Index int
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) String() string {
	var out bytes.Buffer

	out.WriteString("\"" + sl.Value)

	//for i := 0; i < len(sl.Values); i++ {
	//	out.WriteString("[" + sl.Exp[i].Exp.String() + "]")
	//	out.WriteString(sl.Values[i].Value)
	//}

	out.WriteString("\"")

	return out.String()
}
