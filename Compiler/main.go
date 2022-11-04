package main

import (
	"RenG/Compiler/core/compiler"
	"RenG/Compiler/core/lexer"
	"RenG/Compiler/core/parser"
	"RenG/Compiler/file"
	"RenG/Compiler/str"
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

	comp := compiler.New()

	objectTokens := []string{"def", "label"}

	for _, t := range objectTokens {
		oc := str.SliceToken(line, t)
		if oc == "" {
			continue
		}
		l := lexer.New(oc)
		p := parser.New(l)
		pro := p.ParseProgram()
		if len(p.Errors()) != 0 {
			for _, err := range p.Errors() {
				io.WriteString(os.Stdout, err+"\n\n")
			}
			return
		}
		err := comp.Compile(pro)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Compile failed:\n %s\n\n", err)
			return
		}
	}
}

/*
	p := parser.New(l)
	program := p.ParseProgram()
	fmt.Println(program.String())
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
	err = comp.ReplaceSymbol()
	if err != nil {
		fmt.Println("err")
		return
	}

	machine := vm.New(comp.Bytecode())
	err = machine.Run()
	if err != nil {
		fmt.Fprintf(os.Stdout, "Executiong bytecode failed:\n %s\n\n", err)
		return
	}
}
*/
