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
	fmt.Println(obj.(*object.Integer).Value)
}

func testIntegerObject(t *testing.T, obj object.Object, expected int64) bool {
	result, ok := obj.(*object.Integer)
	if !ok {
		t.Errorf("object is not Integer, got=%T (%+v)", obj, obj)
		return false
	}
	if result.Value != expected {
		t.Errorf("object has wrong value. got=%d, want=%d", result.Value, expected)
		return false
	}

	return true
}

func testEval(input string) object.Object {
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	env := object.NewEnvironment()

	return Eval(program, env)
}
