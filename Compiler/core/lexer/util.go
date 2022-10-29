package lexer

import (
	"unicode"
	"unicode/utf8"
)

func readByteUTF8(input string, index int) (r rune, size int) {
	switch {
	case input[index] < 0b11000000:
		return utf8.DecodeRune([]byte{byte(input[index])})
	case input[index] < 0b11100000:
		return utf8.DecodeRune([]byte{byte(input[index]), byte(input[index+1])})
	case input[index] < 0b11110000:
		return utf8.DecodeRune([]byte{byte(input[index]), byte(input[index+1]), byte(input[index+2])})
	case input[index] < 0b11111000:
		return utf8.DecodeRune([]byte{byte(input[index]), byte(input[index+1]), byte(input[index+2]), byte(input[index+3])})
	}
	return
}

func isLetter(ch rune) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || unicode.Is(unicode.Hangul, ch)
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9'
}
