package lexer

import (
	"RenG/interpreter/token"
)

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           byte
}

var (
	inString  = false // [ ] 토큰이 현재 문자열 범위인지 판단하는 역할
	nowString = false // 현재 " " 범위 안에 존재하는 판단하고 [ ] 범위는 문자열이라고 판단한지 못하도록 함
)

func New(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	if !nowString {
		l.skipWhiteSpace()
	}

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.EQ, Literal: literal}
		} else {
			tok = newToken(token.ASSIGN, l.ch)
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.NOT_EQ, Literal: literal}
		} else {
			tok = newToken(token.BANG, l.ch)
		}
	case '+':
		if l.peekChar() == '+' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.PLUS_PLUS, Literal: literal}
		} else if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.PLUS_ASSIGN, Literal: literal}
		} else {
			tok = newToken(token.PLUS, l.ch)
		}
	case '-':
		if l.peekChar() == '-' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.MINUS_MINUS, Literal: literal}
		} else if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.MINUS_ASSIGN, Literal: literal}
		} else {
			tok = newToken(token.MINUS, l.ch)
		}
	case '/':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.SLASH_ASSIGN, Literal: literal}
		} else {
			tok = newToken(token.SLASH, l.ch)
		}
	case '*':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.ASTERISK_ASSIGN, Literal: literal}
		} else {
			tok = newToken(token.ASTERISK, l.ch)
		}
	case '%':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.REMAINDER_ASSIGN, Literal: literal}
		} else {
			tok = newToken(token.REMAINDER, l.ch)
		}
	case '"':
		if nowString {
			tok.Literal = ""
			tok.Type = token.STRING
			l.readChar()
			nowString = false
			return tok
		}
		nowString = true
		tok.Type = token.STRING
		if l.peekChar() == '[' {
			tok.Literal = ""
		} else {
			tok.Literal = l.readString()
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.LT_EQ, Literal: literal}
		} else {
			tok = newToken(token.LT, l.ch)
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.GT_EQ, Literal: literal}
		} else {
			tok = newToken(token.GT, l.ch)
		}
	case ',':
		tok = newToken(token.COMMA, l.ch)
	case '(':
		tok = newToken(token.LPAREN, l.ch)
	case ')':
		tok = newToken(token.RPAREN, l.ch)
	case '{':
		if l.peekChar() == 13 {
			tok = newToken(token.LBRACE, l.ch)
			l.jumpWhiteSpace()
		} else {
			tok = newToken(token.LBRACE, l.ch)
		}
	case '}':
		tok = newToken(token.RBRACE, l.ch)
	case '[':
		if nowString {
			inString = true
			nowString = false
		}
		tok = newToken(token.LBRACKET, l.ch)
	case ']':
		if inString {
			inString = false
			nowString = true
		}
		tok = newToken(token.RBRACKET, l.ch)
	case '\n':
		tok = newToken(token.ENDSENTENCE, l.ch)
	case ';':
		tok = newToken(token.ENDSENTENCE, l.ch)
	case '#':
		for !(l.peekChar() == 13) {
			l.readChar()
		}
		l.readChar()
		tok = newToken(token.ENDSENTENCE, l.ch)
	case 0:
		tok.Literal = ""
		tok.Type = token.EOF
	default:
		if nowString {
			tok.Literal = l.readString()
			tok.Type = token.STRING
			l.readChar()
			return tok
		}
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
			tok = newToken(token.ILLEGAL, l.ch)
		}
	}

	l.readChar()
	return tok
}

// 새로운 토큰을 생성합니다.
func newToken(tokenType token.TokenType, ch byte) token.Token {
	return token.Token{Type: tokenType, Literal: string(ch)}
}

// 문자 하나를 읽고, readPosition,position 모두 +1 증가시킵니다.
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
		l.position = l.readPosition
		l.readPosition += 1
	}
}

// 현재 해당하는 position보다 +1 되어 있는 문자를 가르킵니다.
//     이를 통해 +=, -= 과 같이 2글자로 된 연산자를 렉싱할 수 있게 됩니다.
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	} else {
		return l.input[l.readPosition]
	}
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
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

// 문자열을 모두 읽어들입니다.
func (l *Lexer) readString() string {
	position := l.position + 1

	if l.ch != '"' {
		position--
	}

	for {
		if l.peekChar() == '[' {
			return l.input[position : l.position+1]
		}
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
	}

	nowString = false

	return l.input[position:l.position]
}

// 문자열에 해당하는지 판단
func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

// 정수인지 판단
func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// 화이트 스페이스를 모두 스킵합니다.
//     ' ', \t, \r
func (l *Lexer) skipWhiteSpace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}
}

// \n 개행문자를 스킵하는 역할입니다.
//     필요성 : 개행문자를 하나의 ENDSENTENCE 토큰으로 판단하므로 필요합니다.
func (l *Lexer) jumpWhiteSpace() {
	for l.peekChar() == 13 || l.ch == 13 {
		l.readChar()
	}
	l.readChar()
}
