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
	FONT   = "FONT"

	TEXT        = "TEXT"
	IMAGEBUTTON = "IMAGEBUTTON"
	TEXTBUTTON  = "TEXTBUTTON"
	KEY         = "KEY"
	ACTION      = "ACTION"
	LIMITWIDTH  = "LIMITWIDTH"

	MENU = "MENU"
	CALL = "CALL"
	JUMP = "JUMP"

	IMAGE = "IMAGE"

	CHARACTER = "CHARACTER"

	TRANSFORM  = "TRANSFORM"
	TRANSITION = "TRANSITION"
	STYLE      = "STYLE"

	SCENE = "SCENE"

	SHOW = "SHOW"
	HIDE = "HIDE"

	AT   = "AT"
	AS   = "AS"
	WITH = "WITH"

	PLAY = "PLAY"
	STOP = "STOP"
)

// only transform
const (
	XPOS = "XPOS"
	YPOS = "YPOS"

	XSIZE = "XSIZE"
	YSIZE = "YSIZE"

	ROTATE = "ROTATE"
	ALPHA  = "ALPHA"
)

const (
	COLOR = "COLOR"
)

var keywords = map[string]TokenType{
	"def":         FUNCTION,
	"true":        TRUE,
	"false":       FALSE,
	"if":          IF,
	"elif":        ELIF,
	"else":        ELSE,
	"return":      RETURN,
	"for":         FOR,
	"while":       WHILE,
	"label":       LABEL,
	"font":        FONT,
	"menu":        MENU,
	"call":        CALL,
	"jump":        JUMP,
	"screen":      SCREEN,
	"text":        TEXT,
	"imagebutton": IMAGEBUTTON,
	"textbutton":  TEXTBUTTON,
	"key":         KEY,
	"action":      ACTION,
	"limitWidth":  LIMITWIDTH,
	"image":       IMAGE,
	"character":   CHARACTER,
	"transform":   TRANSFORM,
	"transition":  TRANSITION,
	"style":       STYLE,
	"scene":       SCENE,
	"show":        SHOW,
	"hide":        HIDE,
	"at":          AT,
	"as":          AS,
	"with":        WITH,
	"xpos":        XPOS,
	"ypos":        YPOS,
	"xsize":       XSIZE,
	"ysize":       YSIZE,
	"rotate":      ROTATE,
	"alpha":       ALPHA,
	"color":       COLOR,
	"play":        PLAY,
	"stop":        STOP,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
