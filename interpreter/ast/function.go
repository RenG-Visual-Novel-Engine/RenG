package ast

import (
	"RenG/interpreter/token"
	"bytes"
	"strings"
)

type FunctionExpression struct {
	Token      token.Token
	Name       *Identifier
	Parameters []*Identifier
	Body       *BlockStatement
}

func (fe *FunctionExpression) expressionNode()      {}
func (fe *FunctionExpression) TokenLiteral() string { return fe.Token.Literal }
func (fe *FunctionExpression) String() string {
	var out bytes.Buffer

	params := []string{}
	for _, p := range fe.Parameters {
		params = append(params, p.String())
	}

	out.WriteString(fe.TokenLiteral() + " ")
	out.WriteString(fe.Name.String())
	out.WriteString("(")
	out.WriteString(strings.Join(params, ","))
	out.WriteString(") {\n")
	out.WriteString(fe.Body.String())
	out.WriteString("\n}")

	return out.String()
}
