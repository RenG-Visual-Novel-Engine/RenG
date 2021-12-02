package ast

import (
	"RenG/src/lang/token"
	"bytes"
)

type ScreenExpression struct {
	Token token.Token
	Name  *Identifier
	Body  *BlockStatement
}

func (se *ScreenExpression) expressionNode()      {}
func (se *ScreenExpression) TokenLiteral() string { return se.Token.Literal }
func (se *ScreenExpression) String() string {
	var out bytes.Buffer

	out.WriteString("screen ")
	out.WriteString(se.Name.String())
	out.WriteString(se.Body.String())

	return out.String()
}

type TextExpression struct {
	Token     token.Token
	Text      Expression
	Transform *Identifier
}

func (te *TextExpression) expressionNode()      {}
func (te *TextExpression) TokenLiteral() string { return te.Token.Literal }
func (te *TextExpression) String() string {
	return te.Text.String()
}

type ImagebuttonExpression struct {
	Token     token.Token
	MainImage *Identifier
	Transform *Identifier
	Action    Expression
}

func (ie *ImagebuttonExpression) expressionNode()      {}
func (ie *ImagebuttonExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *ImagebuttonExpression) String() string {
	var out bytes.Buffer

	out.WriteString("imagebutton ")
	out.WriteString(ie.MainImage.String())
	out.WriteString(ie.Transform.String())
	out.WriteString(ie.Action.String())

	return out.String()
}

type TextbuttonExpression struct {
	Token     token.Token
	Text      Expression
	Transform *Identifier
	Action    Expression
}

func (te *TextbuttonExpression) expressionNode()      {}
func (te *TextbuttonExpression) TokenLiteral() string { return te.Token.Literal }
func (te *TextbuttonExpression) String() string {
	var out bytes.Buffer

	out.WriteString("textbutton ")
	out.WriteString(te.Text.String())
	out.WriteString(te.Transform.String())
	out.WriteString(te.Action.String())

	return out.String()
}

type KeyExpression struct {
	Token  token.Token
	Key    Expression
	Action Expression
}

func (ke *KeyExpression) expressionNode()      {}
func (ke *KeyExpression) TokenLiteral() string { return ke.Token.Literal }
func (ke *KeyExpression) String() string {
	return "key " + ke.Key.String() + " action " + ke.Action.String()
}

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

type VideoExpression struct {
	Token token.Token
	Name  *Identifier
	Info  map[string]Expression
}

func (ve *VideoExpression) expressionNode()      {}
func (ve *VideoExpression) TokenLiteral() string { return ve.Token.Literal }
func (ve *VideoExpression) String() string {
	var out bytes.Buffer

	out.WriteString("video ")
	out.WriteString(ve.Name.String())

	for key, value := range ve.Info {
		out.WriteString(key + " " + value.String())
	}

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

type XSizeExpression struct {
	Token token.Token
	Value Expression
}

func (xse *XSizeExpression) expressionNode()      {}
func (xse *XSizeExpression) TokenLiteral() string { return xse.Token.Literal }
func (xse *XSizeExpression) String() string {
	return xse.Value.String()
}

type YSizeExpression struct {
	Token token.Token
	Value Expression
}

func (yse *YSizeExpression) expressionNode()      {}
func (yse *YSizeExpression) TokenLiteral() string { return yse.Token.Literal }
func (yse *YSizeExpression) String() string {
	return yse.Value.String()
}

type RotateExpression struct {
	Token token.Token
	Value Expression
}

func (re *RotateExpression) expressionNode()      {}
func (re *RotateExpression) TokenLiteral() string { return re.Token.Literal }
func (re *RotateExpression) String() string {
	return re.Value.String()
}

type AlphaExpression struct {
	Token token.Token
	Value Expression
}

func (ae *AlphaExpression) expressionNode()      {}
func (ae *AlphaExpression) TokenLiteral() string { return ae.Token.Literal }
func (ae *AlphaExpression) String() string {
	return ae.Value.String()
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

type WhoExpression struct {
	Token token.Token
}

func (we *WhoExpression) expressionNode()      {}
func (we *WhoExpression) TokenLiteral() string { return we.Token.Literal }
func (we *WhoExpression) String() string {
	return "who"
}

type WhatExpression struct {
	Token token.Token
}

func (we *WhatExpression) expressionNode()      {}
func (we *WhatExpression) TokenLiteral() string { return we.Token.Literal }
func (we *WhatExpression) String() string {
	return "what"
}
