package parser

import (
	"RenG/interpreter/ast"
	"RenG/interpreter/token"
)

func (p *Parser) parseGroupExpression() ast.Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
}
