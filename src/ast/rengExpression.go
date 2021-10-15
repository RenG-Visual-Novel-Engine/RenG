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

type TransformExpression struct {
	Token token.Token
	Name  *Identifier
	Body  *BlockStatement
}

func (te *TransformExpression) expressionNode()      {}
func (te *TransformExpression) TokenLiteral() string { return te.Token.Literal }
func (te *TransformExpression) String() string {
	return "transform " + te.Name.String() + "{" + te.Body.String() + "}"
}

type ShowExpression struct {
	Token     token.Token
	Name      *Identifier
	Transform *Identifier
}

func (se *ShowExpression) expressionNode()      {}
func (se *ShowExpression) TokenLiteral() string { return se.Token.Literal }
func (se *ShowExpression) String() string {
	return "show " + se.Name.String() + "at" + se.Transform.String()
}

type XPosExpression struct {
	Token token.Token
	Value Expression
}

func (pe *XPosExpression) expressionNode()      {}
func (pe *XPosExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *XPosExpression) String() string {
	return pe.Value.String()
}

type YPosExpression struct {
	Token token.Token
	Value Expression
}

func (pe *YPosExpression) expressionNode()      {}
func (pe *YPosExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *YPosExpression) String() string {
	return pe.Value.String()
}
