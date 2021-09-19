package parser

import (
	"RenG/src/ast"
	"RenG/src/lexer"
	"RenG/src/token"
)

const (
	_ int = iota
	LOWEST
	ASSIGNMENT
	EQUALS
	LESSGREATER
	SUM
	PRODUCT
	PREFIX
	CALL
	INDEX
)

var precedences = map[token.TokenType]int{
	token.ASSIGN:           ASSIGNMENT,
	token.PLUS_ASSIGN:      ASSIGNMENT,
	token.MINUS_ASSIGN:     ASSIGNMENT,
	token.ASTERISK_ASSIGN:  ASSIGNMENT,
	token.SLASH_ASSIGN:     ASSIGNMENT,
	token.REMAINDER_ASSIGN: ASSIGNMENT,
	token.EQ:               EQUALS,
	token.NOT_EQ:           EQUALS,
	token.LT:               LESSGREATER,
	token.GT:               LESSGREATER,
	token.LT_EQ:            LESSGREATER,
	token.GT_EQ:            LESSGREATER,
	token.PLUS:             SUM,
	token.MINUS:            SUM,
	token.PLUS_PLUS:        SUM,
	token.MINUS_MINUS:      SUM,
	token.SLASH:            PRODUCT,
	token.ASTERISK:         PRODUCT,
	token.REMAINDER:        PRODUCT,
	token.LPAREN:           CALL,
	token.LBRACKET:         INDEX,
}

type Parser struct {
	l      *lexer.Lexer
	errors []string

	curToken  token.Token
	peekToken token.Token

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}
	p.nextToken()
	p.nextToken()

	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.FLOAT, p.parseFloatLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.PLUS_PLUS, p.parsePrefixExpression)
	p.registerPrefix(token.MINUS_MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.LPAREN, p.parseGroupExpression)
	p.registerPrefix(token.LABEL, p.parseLabelExpression)
	p.registerPrefix(token.IF, p.parseIfExpression)
	p.registerPrefix(token.FUNCTION, p.parseFunctionExpression)
	p.registerPrefix(token.WHILE, p.parseWhielExpression)
	p.registerPrefix(token.FOR, p.parseForExpression)

	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.SLASH, p.parseInfixExpression)
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
	p.registerInfix(token.REMAINDER, p.parseInfixExpression)
	p.registerInfix(token.EQ, p.parseInfixExpression)
	p.registerInfix(token.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)
	p.registerInfix(token.LT_EQ, p.parseInfixExpression)
	p.registerInfix(token.GT_EQ, p.parseInfixExpression)
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)
	p.registerInfix(token.ASSIGN, p.parseInfixExpression)
	p.registerInfix(token.PLUS_ASSIGN, p.parseInfixExpression)
	p.registerInfix(token.MINUS_ASSIGN, p.parseInfixExpression)
	p.registerInfix(token.ASTERISK_ASSIGN, p.parseInfixExpression)
	p.registerInfix(token.SLASH_ASSIGN, p.parseInfixExpression)
	p.registerInfix(token.REMAINDER_ASSIGN, p.parseInfixExpression)

	return p
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for p.curToken.Type != token.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}
	return program
}
