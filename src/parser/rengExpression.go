package parser

import (
	"RenG/src/ast"
	"RenG/src/token"
)

func (p *Parser) parseLabelExpression() ast.Expression {
	expression := &ast.LabelExpression{Token: p.curToken}

	p.nextToken()

	expression.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	expression.Body = p.parseBlockStatement()

	return expression
}

func (p *Parser) parseImageExpression() ast.Expression {
	exp := &ast.ImageExpression{Token: p.curToken}

	p.nextToken()

	exp.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()

	exp.Path = p.parseStringLiteral()

	return exp
}

func (p *Parser) parseShowExpression() ast.Expression {
	exp := &ast.ShowExpression{Token: p.curToken}

	p.nextToken()

	exp.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	return exp
}
