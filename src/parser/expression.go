package parser

import (
	"RenG/src/ast"
	"RenG/src/token"
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

// [  ,  ,  ] 와 같은 리스트를 파싱합니다
func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	list := []ast.Expression{}

	if p.peekTokenIs(end) {
		p.nextToken()
		return list
	}

	p.nextToken()
	list = append(list, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		list = append(list, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(end) {
		return nil
	}

	return list
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()

	expression.Right = p.parseExpression(PREFIX)

	return expression
}

// 그룹 표현식을 파싱함 ex -> (1 + 2) == 3
func (p *Parser) parseGroupExpression() ast.Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
}

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

// 함수를 파싱합니다.
func (p *Parser) parseFunctionExpression() ast.Expression {
	lit := &ast.FunctionExpression{Token: p.curToken}

	p.nextToken()

	lit.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	lit.Parameters = p.parseFunctionParameters()

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	lit.Body = p.parseBlockStatement()

	return lit
}

func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	identifiers := []*ast.Identifier{}

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return identifiers
	}

	p.nextToken()

	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	identifiers = append(identifiers, ident)

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		identifiers = append(identifiers, ident)
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return identifiers
}

// 함수 호출 표현식입니다.
//     call(args ...)
//         ^--- 해당 소괄호가 키워드입니다.
// 중위 연산자로 취급됩니다.
func (p *Parser) parseCallFunctionExpression(function ast.Expression) ast.Expression {
	exp := &ast.CallFunctionExpression{Token: p.curToken, Function: function}
	exp.Arguments = p.parseCallArguments()
	return exp
}

func (p *Parser) parseCallArguments() []ast.Expression {
	args := []ast.Expression{}

	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return args
	}

	p.nextToken()
	args = append(args, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return args
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	exp := &ast.IndexExpression{Token: p.curToken, Left: left}

	p.nextToken()
	exp.Index = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RBRACKET) {
		return nil
	}

	return exp
}

func (p *Parser) parseWhileExpression() ast.Expression {
	exp := &ast.WhileExpression{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()

	exp.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	exp.Body = p.parseBlockStatement()

	return exp
}

// for 문을 파싱합니다.
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
