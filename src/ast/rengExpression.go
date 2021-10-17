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

type CallLabelExpression struct {
	Token token.Token
	Label *Identifier
}

func (cle *CallLabelExpression) expressionNode()      {}
func (cle *CallLabelExpression) TokenLiteral() string { return cle.Token.Literal }
func (cle *CallLabelExpression) String() string       { return "call " + cle.Label.String() }

type JumpLabelExpression struct {
	Token token.Token
	Label *Identifier
}

func (jle *JumpLabelExpression) expressionNode()      {}
func (jle *JumpLabelExpression) TokenLiteral() string { return jle.Token.Literal }
func (jle *JumpLabelExpression) String() string       { return "jump " + jle.Label.String() }

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

type HideExpression struct {
	Token token.Token
	Name  *Identifier
}

func (he *HideExpression) expressionNode()      {}
func (he *HideExpression) TokenLiteral() string { return he.Token.Literal }
func (he *HideExpression) String() string {
	return "hide " + he.Name.String()
}

type XPosExpression struct {
	Token token.Token
	Value Expression
}

func (xpe *XPosExpression) expressionNode()      {}
func (xpe *XPosExpression) TokenLiteral() string { return xpe.Token.Literal }
func (xpe *XPosExpression) String() string {
	return xpe.Value.String()
}

type YPosExpression struct {
	Token token.Token
	Value Expression
}

func (ype *YPosExpression) expressionNode()      {}
func (ype *YPosExpression) TokenLiteral() string { return ype.Token.Literal }
func (ype *YPosExpression) String() string {
	return ype.Value.String()
}

type PlayExpression struct {
	Token   token.Token
	Channel *Identifier
	Music   Expression
	Loop    *Identifier
}

func (pe *PlayExpression) expressionNode()      {}
func (pe *PlayExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PlayExpression) String() string {
	return "play " + pe.Channel.Value + " " + pe.Music.String()
}

type StopExpression struct {
	Token   token.Token
	Channel *Identifier
}

func (se *StopExpression) expressionNode()      {}
func (se *StopExpression) TokenLiteral() string { return se.Token.Literal }
func (se *StopExpression) String() string {
	return "stop " + se.Channel.String()
}
