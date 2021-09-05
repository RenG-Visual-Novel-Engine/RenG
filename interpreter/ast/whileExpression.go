package ast

import (
	"RenG/interpreter/token"
	"bytes"
)

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
