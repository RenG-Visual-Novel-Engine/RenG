package parser

import (
	"RenG/interpreter/ast"
	"RenG/interpreter/token"
)

func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}
