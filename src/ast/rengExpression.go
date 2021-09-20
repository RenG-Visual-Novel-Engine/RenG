package ast

import (
	"RenG/src/token"
	"bytes"
)

type LabelExpression struct {
	Token token.Token
	Name  *Identifier
	Body  *BlockStatement
}

func (le *LabelExpression) expressionNode()      {}
func (le *LabelExpression) TokenLiteral() string { return le.Token.Literal }
func (le *LabelExpression) String() string {
	var out bytes.Buffer

	out.WriteString("label ")
	out.WriteString(le.Name.String())
	out.WriteString(le.Body.String())

	return out.String()
}
