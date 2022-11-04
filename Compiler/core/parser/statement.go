package parser

import (
	"RenG/Compiler/core/ast"
	"RenG/Compiler/core/token"
)

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	if p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		return &ast.BlockStatement{}
	}

	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	if p.peekTokenIs(token.ENDSENTENCE) {
		p.nextToken()
	}

	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		block.Statements = append(block.Statements, stmt)

		if p.curTokenIs(token.ENDSENTENCE) {
			p.nextToken()
		}
	}

	for p.peekTokenIs(token.ENDSENTENCE) {
		p.nextToken()
	}

	return block
}

func (p *Parser) parseFunctionStatement() ast.Statement {
	stmt := &ast.FunctionStatement{Token: p.curToken}

	p.nextToken()

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	stmt.Parameters = p.parseFunctionParameters()

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
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

func (p *Parser) parseIfStatement() ast.Statement {
	stmt := &ast.IfStatement{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()

	stmt.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	stmt.Consequence = p.parseBlockStatement()

	for p.peekTokenIs(token.ELIF) {
		p.nextToken()

		elifStmt := &ast.IfStatement{Token: p.curToken}

		if !p.expectPeek(token.LPAREN) {
			return nil
		}

		p.nextToken()

		elifStmt.Condition = p.parseExpression(LOWEST)

		if !p.expectPeek(token.RPAREN) {
			return nil
		}

		if !p.expectPeek(token.LBRACE) {
			return nil
		}

		elifStmt.Consequence = p.parseBlockStatement()

		stmt.Elif = append(stmt.Elif, elifStmt)
	}

	if p.peekTokenIs(token.ELSE) {
		p.nextToken()

		if !p.expectPeek(token.LBRACE) {
			return nil
		}

		stmt.Else = p.parseBlockStatement()
	}

	if p.curTokenIs(token.RBRACE) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseForStatement() *ast.ForStatement {
	stmt := &ast.ForStatement{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()

	if p.curTokenIs(token.ENDSENTENCE) {
		stmt.Initialization = nil
		p.ENDSENTENCETokenSkip()
	} else {
		p.ENDSENTENCETokenSkip()
		stmt.Initialization = p.parseExpression(LOWEST)
		p.ENDSENTENCETokenSkip()
	}

	if p.curTokenIs(token.ENDSENTENCE) {
		stmt.Condition = nil
		p.ENDSENTENCETokenSkip()
	} else {
		p.ENDSENTENCETokenSkip()
		stmt.Condition = p.parseExpression(LOWEST)
		p.ENDSENTENCETokenSkip()
	}

	if p.curTokenIs(token.RPAREN) {
		stmt.Increment = nil
	} else {
		p.ENDSENTENCETokenSkip()
		stmt.Increment = p.parseExpression(LOWEST)
		p.nextToken()
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	stmt.Loop = p.parseBlockStatement()

	if p.curTokenIs(token.RBRACE) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken()

	if p.curToken.Type == token.ENDSENTENCE {
		return stmt
	}

	stmt.ReturnValue = p.parseExpression(LOWEST)

	for p.peekTokenIs(token.ENDSENTENCE) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseScreenStatement() *ast.ScreenStatement {
	stmt := &ast.ScreenStatement{Token: p.curToken}

	p.nextToken()

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}
