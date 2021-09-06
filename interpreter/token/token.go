package token

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// 변수
	IDENT = "IDENT"

	// 정수
	INT = "INT"

	// 연산자
	ASSIGN    = "="
	PLUS      = "+"
	MINUS     = "-"
	BANG      = "!"
	ASTERISK  = "*"
	SLASH     = "/"
	REMAINDER = "%"

	// 불 연산
	EQ     = "=="
	NOT_EQ = "!="

	LT = "<"
	GT = ">"

	LT_EQ = "<="
	GT_EQ = ">="

	// 구분자
	COMMA       = ","
	ENDSENTENCE = "ENDSENTENCE"

	LPAREN = "("
	RPAREN = ")"
	LBRACE = "{"
	RBRACE = "}"

	// 주석
	COMMENT = "#"

	// 반복문
	FOR   = "FOR"
	WHILE = "WHILE"

	// 예약어
	FUNCTION = "FUNCTION"
	SCREEN   = "SCREEN"
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	IF       = "IF"
	ELIF     = "ELIF"
	ELSE     = "ELSE"
	RETURN   = "RETURN"
	LABEL    = "LABEL"
)

var keywords = map[string]TokenType{
	"def":    FUNCTION,
	"screen": SCREEN,
	"true":   TRUE,
	"false":  FALSE,
	"if":     IF,
	"elif":   ELIF,
	"else":   ELSE,
	"return": RETURN,
	"for":    FOR,
	"while":  WHILE,
	"label":  LABEL,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
