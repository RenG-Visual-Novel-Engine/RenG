package main

import (
	"RenG/Compiler/lexer"
	"RenG/Compiler/parser"
	"fmt"
)

func main() {
	test := `1 + 4++ + 6 * 8-- / ++10`
	l := lexer.New(test)
	p := parser.New(l)
	fmt.Println(p.ParseProgram().String())
}
