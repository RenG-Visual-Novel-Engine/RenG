package parser

import (
	"RenG/interpreter/ast"
	"RenG/interpreter/token"
)

func (p *Parser) parseIfExpression() ast.Expression {
	expression := &ast.IfExpression{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()

	expression.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	expression.Consequence = p.parseBlockStatement()

	for p.peekTokenIs(token.ELIF) {
		p.nextToken()

		elifExpression := &ast.ElifExpression{Token: p.curToken}

		if !p.expectPeek(token.LPAREN) {
			return nil
		}

		p.nextToken()

		elifExpression.Condition = p.parseExpression(LOWEST)

		if !p.expectPeek(token.RPAREN) {
			return nil
		}

		if !p.expectPeek(token.LBRACE) {
			return nil
		}

		elifExpression.Consequence = p.parseBlockStatement()

		expression.Elif = append(expression.Elif, elifExpression)
	}

	if p.peekTokenIs(token.ELSE) {
		p.nextToken()

		if !p.expectPeek(token.LBRACE) {
			return nil
		}

		expression.Alternative = p.parseBlockStatement()
	}

	return expression
}
