package parser

import (
	"RenG/interpreter/lexer"
	"io/ioutil"
	"testing"
)

func TestLebelExpression(t *testing.T) {
	code, err := ioutil.ReadFile("main.rgo")
	if err != nil {
		panic(err)
	}

	l := lexer.New(string(code))
	p := New(l)
	program := p.ParseProgram()

	program.String()
}
