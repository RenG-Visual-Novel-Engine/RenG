package parser

import (
	"RenG/interpreter/ast"
	"RenG/interpreter/token"
)

// 표현식을 파싱합니다.
// ex)
//     1 + 10 + 11 => ((1 + 10) + 11)
func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	// token.RBRACKET(']')이 ENDSENTENCE 취급 받는 것은 "[Expression]" 과 같은 문법을 지원하기 위해서 입니다.
	for !(p.peekTokenIs(token.ENDSENTENCE) || p.peekTokenIs(token.RBRACKET)) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}
	stmt.Expression = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.ENDSENTENCE) {
		p.nextToken()
	}

	return stmt
}
