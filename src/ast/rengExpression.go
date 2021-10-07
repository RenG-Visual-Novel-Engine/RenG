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

type ImageExpression struct {
	Token token.Token
	Name  *Identifier
	Path  Expression
}

func (ie *ImageExpression) expressionNode()      {}
func (ie *ImageExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *ImageExpression) String() string {
	var out bytes.Buffer

	out.WriteString("image")
	out.WriteString(ie.Name.String())
	out.WriteString(ie.Path.String())

	return out.String()
}

type ShowExpression struct {
	Token token.Token
	Name  *Identifier
}

func (se *ShowExpression) expressionNode()      {}
func (se *ShowExpression) TokenLiteral() string { return se.Token.Literal }
func (se *ShowExpression) String() string       { return "show " + se.Name.String() }
