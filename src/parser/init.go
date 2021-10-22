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
	OR_BOOL
	AND_BOOL
	OR
	XOR
	AND
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
	token.OR_BOOL:          OR_BOOL,
	token.AND_BOOL:         AND_BOOL,
	token.OR:               OR,
	token.XOR:              XOR,
	token.AND:              AND,
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
	p.registerPrefix(token.IF, p.parseIfExpression)
	p.registerPrefix(token.FUNCTION, p.parseFunctionExpression)
	p.registerPrefix(token.WHILE, p.parseWhileExpression)
	p.registerPrefix(token.FOR, p.parseForExpression)
	p.registerPrefix(token.LABEL, p.parseLabelExpression)
	p.registerPrefix(token.CALL, p.parseCallLabelExpression)
	p.registerPrefix(token.JUMP, p.parseJumpLabelExpression)
	p.registerPrefix(token.IMAGE, p.parseImageExpression)
	p.registerPrefix(token.VIDEO, p.parseVideoExpression)
	p.registerPrefix(token.SHOW, p.parseShowExpression)
	p.registerPrefix(token.HIDE, p.parseHideExpression)
	p.registerPrefix(token.TRANSFORM, p.parseTranformExpression)
	p.registerPrefix(token.XPOS, p.parseXposExpression)
	p.registerPrefix(token.YPOS, p.parseYposExpression)
	p.registerPrefix(token.PLAY, p.parsePlayExpression)
	p.registerPrefix(token.STOP, p.parseStopExpression)

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
	p.registerInfix(token.LPAREN, p.parseCallFunctionExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)
	p.registerInfix(token.AND_BOOL, p.parseInfixExpression)
	p.registerInfix(token.OR_BOOL, p.parseInfixExpression)
	p.registerInfix(token.OR, p.parseInfixExpression)
	p.registerInfix(token.XOR, p.parseInfixExpression)
	p.registerInfix(token.AND, p.parseInfixExpression)
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
		program.Statements = append(program.Statements, stmt)
		p.nextToken()
	}
	return program
}
