package parser

import (
	"RenG/interpreter/lexer"
	"fmt"
	"io/ioutil"
	"testing"
)

func TestLabelExpression(t *testing.T) {
	code, err := ioutil.ReadFile("main.rgo")
	if err != nil {
		panic(err)
	}

	l := lexer.New(string(code))
	p := New(l)
	program := p.ParseProgram()

	fmt.Println(program.String())
}
