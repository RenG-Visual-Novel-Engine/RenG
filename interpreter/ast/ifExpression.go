package ast

import (
	"RenG/interpreter/token"
	"bytes"
)

type IfExpression struct {
	Token       token.Token
	Condition   Expression
	Consequence *BlockStatement
	Elif        []*ElifExpression
	Alternative *BlockStatement
}

type ElifExpression struct {
	Token       token.Token
	Condition   Expression
	Consequence *BlockStatement
}

func (ie *IfExpression) expressionNode()      {}
func (ie *IfExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IfExpression) String() string {
	var out bytes.Buffer

	out.WriteString("if " + ie.Condition.String())
	out.WriteString(" {\n" + ie.Consequence.String() + "\n}")

	for _, ee := range ie.Elif {
		if ee != nil {
			out.WriteString("elif ")
			out.WriteString(ee.Condition.String())
			out.WriteString(" ")
			out.WriteString(ie.Consequence.String())
		}
	}

	if ie.Alternative != nil {
		out.WriteString("else ")
		out.WriteString(ie.Alternative.String())
	}

	return out.String()
}
