package parser

import (
	"RenG/interpreter/ast"
	"RenG/interpreter/token"
)

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.RETURN:
		return p.parseReturnStatement()
	case token.VAR:
		return p.parseVarStatement()
	default:
		return p.parseExpressionStatement()
	}
}
