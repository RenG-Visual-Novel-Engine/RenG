package parser

import (
	"fmt"

	"RenG/interpreter/token"
)

/*---------------error----------------*/

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %s found", t)
	p.errors = append(p.errors, msg)
}

func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instend", t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

/*-------------token---------------*/

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIS(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIS(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}
