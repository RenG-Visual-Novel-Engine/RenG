package token

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
}

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// 변수
	IDENT = "IDENT"

	// 타입
	INT    = "INT"
	FLOAT  = "FLOAT"
	STRING = "STRING"

	// 연산자
	ASSIGN           = "="
	PLUS_ASSIGN      = "+="
	MINUS_ASSIGN     = "-="
	ASTERISK_ASSIGN  = "*="
	SLASH_ASSIGN     = "/="
	REMAINDER_ASSIGN = "%="

	PLUS_PLUS   = "++"
	MINUS_MINUS = "--"

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

	AND_BOOL = "&&"
	OR_BOOL  = "||"

	// 비트 연산
	AND = "&"
	OR  = "|"
	XOR = "^"

	// 구분자
	COMMA       = ","
	ENDSENTENCE = "ENDSENTENCE"

	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"

	// 주석
	COMMENT = "#"

	// 반복문
	FOR   = "FOR"
	WHILE = "WHILE"

	// 예약어
	FUNCTION = "FUNCTION"
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	IF       = "IF"
	ELIF     = "ELIF"
	ELSE     = "ELSE"
	RETURN   = "RETURN"
)

const (
	LABEL  = "LABEL"
	SCREEN = "SCREEN"

	CALL = "CALL"
	JUMP = "JUMP"

	IMAGE     = "IMAGE"
	VIDEO     = "VIDEO"
	CHARACTER = "CHARACTER"
	TRANSFORM = "TRANSFORM"

	SCENE = "SCENE"

	SHOW = "SHOW"
	HIDE = "HIDE"
	AT   = "AT"

	PLAY = "PLAY"
	STOP = "STOP"
)

// only transform
const (
	XPOS = "XPOS"
	YPOS = "YPOS"
)

var keywords = map[string]TokenType{
	"def":       FUNCTION,
	"true":      TRUE,
	"false":     FALSE,
	"if":        IF,
	"elif":      ELIF,
	"else":      ELSE,
	"return":    RETURN,
	"for":       FOR,
	"while":     WHILE,
	"label":     LABEL,
	"screen":    SCREEN,
	"call":      CALL,
	"jump":      JUMP,
	"image":     IMAGE,
	"video":     VIDEO,
	"character": CHARACTER,
	"transform": TRANSFORM,
	"scene":     SCENE,
	"show":      SHOW,
	"hide":      HIDE,
	"at":        AT,
	"xpos":      XPOS,
	"ypos":      YPOS,
	"play":      PLAY,
	"stop":      STOP,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
