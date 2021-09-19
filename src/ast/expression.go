package ast

import (
	"RenG/src/token"
	"bytes"
	"strings"
)

/*
 * Prefix Expression Node
 * 전위 연산자 표현식
 */
type PrefixExpression struct {
	Token    token.Token
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(pe.Operator)
	out.WriteString(pe.Right.String())
	out.WriteString(")")

	return out.String()
}

/*
 * Infix Expression Node
 * 중위 연산자 표현식
 */
type InfixExpression struct {
	Token    token.Token
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *InfixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString(" " + ie.Operator + " ")
	out.WriteString(ie.Right.String())
	out.WriteString(")")

	return out.String()
}

/*
 * If Expression Node
 */
type IfExpression struct {
	Token       token.Token
	Condition   Expression
	Consequence *BlockStatement
	Elif        []*ElifExpression
	Alternative *BlockStatement
}

/*
 * Elif Expression Node
 */
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

/*
 * Function Expression Node
 */
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

/*
 * Call Exprssion Node
 */
type CallExpression struct {
	Token     token.Token
	Function  Expression
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	var out bytes.Buffer

	args := []string{}
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}

	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ","))
	out.WriteString(")")

	return out.String()
}

/*
 * Index Expressio Node
 */
type IndexExpression struct {
	Token token.Token
	Left  Expression
	Index Expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString("[")
	out.WriteString(ie.Index.String())
	out.WriteString("]")

	return out.String()
}

/*
 * While Exprssion Node
 */
type WhileExpression struct {
	Token     token.Token
	Condition Expression
	Body      *BlockStatement
}

func (we *WhileExpression) expressionNode()      {}
func (we *WhileExpression) TokenLiteral() string { return we.Token.Literal }
func (we *WhileExpression) String() string {
	var out bytes.Buffer

	out.WriteString("while " + we.Condition.String() + "\n{" + we.Body.String() + "\n}")

	return out.String()
}

/*
 * For Expression Node
 */
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
