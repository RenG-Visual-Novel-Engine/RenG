package main

import (
	"fmt"
	"io/ioutil"

	"RenG/interpreter/lexer"
	"RenG/interpreter/token"
)

func main() {
	code, err := ioutil.ReadFile("main.rgo")
	if err != nil {
		panic(err)
	}

	l := lexer.New(string(code))

	for i := l.NextToken(); i.Type != token.EOF; i = l.NextToken() {
		fmt.Println(i)
	}
}
