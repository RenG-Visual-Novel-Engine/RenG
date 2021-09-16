package parser

import (
	"RenG/interpreter/ast"
	"RenG/interpreter/token"
)

// bool 오브젝트를 반환합닏.
func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}
