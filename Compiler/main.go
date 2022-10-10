package main

import (
	"RenG/Compiler/compiler"
	"RenG/Compiler/file"
	"RenG/Compiler/lexer"
	"RenG/Compiler/parser"
	"RenG/Compiler/vm"
	"fmt"
	"io"
	"os"
)

/*
	func main() {
		if len(os.Args) != 3 {
			return
		}

		rf := file.CreateFile(os.Args[1])
		defer rf.CloseFile()

		RgoCode := rf.Read()

		l := lexer.New(RgoCode)
		p := parser.New(l)

		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			for _, err := range p.Errors() {
				fmt.Println(err)
			}
			return
		}

		comp := compiler.New()
		err := comp.Compile(program)
		if err != nil {
			fmt.Println(err)
			return
		}

		wf := file.CreateFile(os.Args[2])
		defer wf.CloseFile()

		wf.WriteConstant(comp.Bytecode().Constants)
		wf.WriteInstruction(comp.Bytecode().Instructions, comp.Bytecode().Constants)
	}
*/

func main() {

	f := file.CreateFile("D:\\program\\Go\\src\\RenG\\test\\Test2\\main.rgo")
	line := f.Read()
	l := lexer.New(line)
	p := parser.New(l)

	program := p.ParseProgram()
	// fmt.Println(program.String())
	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			io.WriteString(os.Stdout, err+"\n\n")
		}
		return
	}

	comp := compiler.New()
	err := comp.Compile(program)
	if err != nil {
		fmt.Fprintf(os.Stdout, "Compile failed:\n %s\n\n", err)
		return
	}
	fmt.Println(comp)

	machine := vm.New(comp.Bytecode())
	err = machine.Run()
	if err != nil {
		fmt.Fprintf(os.Stdout, "Executiong bytecode failed:\n %s\n\n", err)
		return
	}

	stackTop := machine.LastPoppedStackElem()
	io.WriteString(os.Stdout, stackTop.Inspect())
	io.WriteString(os.Stdout, "\n")
}
