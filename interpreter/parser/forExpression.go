package parser

import (
	"RenG/interpreter/ast"
	"RenG/interpreter/token"
)

// for 문을 파싱했습니다.
// ex)
//    for (<Expression>; <Expression>; <Expression>) { ... }
func (p *Parser) parseForExpression() ast.Expression {
	exp := &ast.ForExpression{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()

	exp.Define = p.parseExpression(LOWEST)

	if !p.expectPeek(token.ENDSENTENCE) {
		return nil
	}
	p.nextToken()

	exp.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.ENDSENTENCE) {
		return nil
	}
	p.nextToken()

	exp.Run = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	exp.Body = p.parseBlockStatement()

	return exp
}
