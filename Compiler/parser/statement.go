package parser

import (
	"RenG/Compiler/ast"
	"RenG/Compiler/token"
)

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	if p.peekTokenIs(token.RBRACE) {
		p.nextToken()
		return &ast.BlockStatement{}
	}

	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		block.Statements = append(block.Statements, stmt)

		if p.curTokenIs(token.ENDSENTENCE) {
			p.nextToken()
		}
	}

	return block
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

	return stmt
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken()

	if p.curToken.Type == token.ENDSENTENCE {
		return stmt
	}

	stmt.ReturnValue = p.parseExpression(LOWEST)

	for !(p.curTokenIs(token.ENDSENTENCE) || p.curTokenIs(token.RBRACE)) {
		p.nextToken()
	}

	p.nextToken()

	return stmt
}
