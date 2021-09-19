package evaluator

import (
	"RenG/interpreter/lexer"
	"RenG/interpreter/object"
	"RenG/interpreter/parser"
	"fmt"
	"io/ioutil"
	"testing"
)

func TestFunction(t *testing.T) {
	code, err := ioutil.ReadFile("main.rgo")
	if err != nil {
		panic(err)
	}

	obj := testEval(string(code))
	if err, ok := obj.(*object.Error); ok {
		fmt.Println(err.Inspect())
	}
}

func testEval(input string) object.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := object.NewEnvironment()

	return Eval(program, env)
}
