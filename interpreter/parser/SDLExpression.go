package parser

import (
	"RenG/interpreter/ast"
	"RenG/interpreter/token"
)

func (p *Parser) parseLabelExpression() ast.Expression {
	expression := &ast.LabelExpression{Token: p.curToken}

	p.nextToken()

	expression.Name = p.parseExpression(LOWEST)

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	expression.Body = p.parseBlockStatement()

	return expression
}
