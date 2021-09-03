package main

import (
	"io/ioutil"

	"github.com/alvin1007/RenG/interpreter/lexer"
	"github.com/alvin1007/RenG/interpreter/parser"
)

func main() {
	code, err := ioutil.ReadFile("main.rgo")
	if err != nil {
		panic(err)
	}

	l := lexer.New(string(code))
	p := parser.New(l)
	program := p.ParseProgram()

	program.String()
}
