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
	VIDEO = "VIDEO"

	CHARACTER = "CHARACTER"

	TRANSFORM = "TRANSFORM"
	STYLE     = "STYLE"

	SCENE = "SCENE"

	SHOW = "SHOW"
	HIDE = "HIDE"

	AT = "AT"
	AS = "AS"

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

// only style
const (
	COLOR        = "COLOR"
	TYPINGEFFECT = "TYPINGEFFECT"
)

const (
	WHAT = "WHAT"
	WHO  = "WHO"

	ITEMS = "ITEMS"
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
	"video":       VIDEO,
	"character":   CHARACTER,
	"transform":   TRANSFORM,
	"style":       STYLE,
	"scene":       SCENE,
	"show":        SHOW,
	"hide":        HIDE,
	"at":          AT,
	"as":          AS,
	"xpos":        XPOS,
	"ypos":        YPOS,
	"xsize":       XSIZE,
	"ysize":       YSIZE,
	"rotate":      ROTATE,
	"alpha":       ALPHA,
	"color":       COLOR,
	"typing":      TYPINGEFFECT,
	"play":        PLAY,
	"stop":        STOP,
	"who":         WHO,
	"what":        WHAT,
	"items":       ITEMS,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
