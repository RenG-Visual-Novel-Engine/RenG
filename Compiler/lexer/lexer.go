package lexer

import "RenG/Compiler/token"

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           rune
}

func New(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	if l.ch == '\n' {
		l.skipWhiteSpace()
		return newToken(token.ENDSENTENCE, "ENDSENTENCE")
	}

	l.skipWhiteSpace()

	switch l.ch {
	case '=':
		switch l.peekChar() {
		case '=':
			literal := l.ch
			l.readChar()
			tok = newToken(token.EQ, string(literal)+string(l.ch))
		default:
			tok = newToken(token.ASSIGN, string(l.ch))
		}
	case '!':
		switch l.peekChar() {
		case '=':
			literal := l.ch
			l.readChar()
			tok = newToken(token.NOT_EQ, string(literal)+string(l.ch))
		default:
			tok = newToken(token.BANG, string(l.ch))
		}
	case '+':
		switch l.peekChar() {
		case '+':
			literal := l.ch
			l.readChar()
			tok = newToken(token.PLUS_PLUS, string(literal)+string(l.ch))
		case '=':
			literal := l.ch
			l.readChar()
			tok = newToken(token.PLUS_ASSIGN, string(literal)+string(l.ch))
		default:
			tok = newToken(token.PLUS, string(l.ch))
		}
	case '-':
		switch l.peekChar() {
		case '-':
			literal := l.ch
			l.readChar()
			tok = newToken(token.MINUS_MINUS, string(literal)+string(l.ch))
		case '=':
			literal := l.ch
			l.readChar()
			tok = newToken(token.MINUS_ASSIGN, string(literal)+string(l.ch))
		default:
			tok = newToken(token.MINUS, string(l.ch))
		}
	case '/':
		switch l.peekChar() {
		case '=':
			literal := l.ch
			l.readChar()
			tok = newToken(token.SLASH_ASSIGN, string(literal)+string(l.ch))
		default:
			tok = newToken(token.SLASH, string(l.ch))
		}
	case '*':
		switch l.peekChar() {
		case '=':
			literal := l.ch
			l.readChar()
			tok = newToken(token.ASTERISK_ASSIGN, string(literal)+string(l.ch))
		default:
			tok = newToken(token.ASTERISK, string(l.ch))
		}
	case '%':
		switch l.peekChar() {
		case '=':
			literal := l.ch
			l.readChar()
			tok = newToken(token.REMAINDER_ASSIGN, string(literal)+string(l.ch))
		default:
			tok = newToken(token.REMAINDER, string(l.ch))
		}
	case '"':
		tok.Type = token.STRING
		tok.Literal = l.readString()
	case '<':
		switch l.peekChar() {
		case '=':
			literal := l.ch
			l.readChar()
			tok = newToken(token.LT_EQ, string(literal)+string(l.ch))
		default:
			tok = newToken(token.LT, string(l.ch))
		}
	case '>':
		switch l.peekChar() {
		case '=':
			literal := l.ch
			l.readChar()
			tok = newToken(token.GT_EQ, string(literal)+string(l.ch))
		default:
			tok = newToken(token.GT, string(l.ch))
		}
	case '&':
		switch l.peekChar() {
		case '=':
			literal := l.ch
			l.readChar()
			tok = newToken(token.AND_BOOL, string(literal)+string(l.ch))
		default:
			tok = newToken(token.AND, string(l.ch))
		}
	case '|':
		switch l.peekChar() {
		case '=':
			literal := l.ch
			l.readChar()
			tok = newToken(token.OR_BOOL, string(literal)+string(l.ch))
		default:
			tok = newToken(token.OR, string(l.ch))
		}
	case '^':
		tok = newToken(token.XOR, string(l.ch))
	case ',':
		tok = newToken(token.COMMA, string(l.ch))
	case '(':
		tok = newToken(token.LPAREN, string(l.ch))
	case ')':
		tok = newToken(token.RPAREN, string(l.ch))
	case '{':
		tok = newToken(token.LBRACE, string(l.ch))
		l.skipWhiteSpace()
	case '}':
		tok = newToken(token.RBRACE, string(l.ch))
	case '[':
		tok = newToken(token.LBRACKET, string(l.ch))
	case ']':
		tok = newToken(token.RBRACKET, string(l.ch))
	case ';':
		tok = newToken(token.ENDSENTENCE, string(l.ch))
	case '#':
		for !(l.peekChar() == '\n') {
			l.readChar()
		}
		l.skipWhiteSpace()
	case 0:
		tok.Literal = ""
		tok.Type = token.EOF
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			if Literal, ok := l.readNumberAndIsInt(); ok {
				tok.Type = token.INT
				tok.Literal = Literal
				return tok
			} else {
				tok.Type = token.FLOAT
				tok.Literal = Literal
				return tok
			}
		} else {
			tok = newToken(token.ILLEGAL, string(l.ch))
		}
	}

	l.readChar()
	return tok
}

func newToken(tokenType token.TokenType, ch string) token.Token {
	return token.Token{Type: tokenType, Literal: ch}
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
		l.position = l.readPosition
		l.readPosition += 1
	} else {
		r, size := readByteUTF8(l.input, l.readPosition)
		l.ch = r
		l.position = l.readPosition
		l.readPosition += size
	}
}

func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	} else {
		r, _ := readByteUTF8(l.input, l.readPosition)
		return r
	}
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readString() string {
	position := l.readPosition

	for l.peekChar() != '"' || l.peekChar() != 0 {
		l.readChar()
	}
	l.readChar()

	return l.input[position:l.position]
}

func (l *Lexer) readNumberAndIsInt() (string, bool) {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	if l.ch == '.' {
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
		return l.input[position:l.position], false
	}
	return l.input[position:l.position], true
}

func (l *Lexer) skipWhiteSpace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' {
		l.readChar()
	}
}
