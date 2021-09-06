package ast

import (
	"RenG/interpreter/token"
	"bytes"
)

type ForExpression struct {
	Token     token.Token
	Define    Expression
	Condition Expression
	Run       Expression
	Body      *BlockStatement
}

func (fe *ForExpression) expressionNode()      {}
func (fe *ForExpression) TokenLiteral() string { return fe.Token.Literal }
func (fe *ForExpression) String() string {
	var out bytes.Buffer

	out.WriteString("for " + "(" + fe.Define.String() + "; " + fe.Condition.String() + "; " + fe.Run.String() + ")" + " {\n" + fe.Body.String() + "\n}")

	return out.String()
}
