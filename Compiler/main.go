package main

import (
	"RenG/Compiler/compiler"
	"RenG/Compiler/file"
	"RenG/Compiler/lexer"
	"RenG/Compiler/parser"
	"RenG/Compiler/util"
	"fmt"
	"io"
	"os"
)

func init() {
	for _, arg := range os.Args[1:] {
		switch arg {
		}
	}
}

func main() {
	f := file.CreateFile("./test.rgoc")
	defer f.CloseFile()

	testCode := "1 + 1 * 5"

	l := lexer.New(testCode)
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		for _, err := range p.Errors() {
			io.WriteString(os.Stdout, err)
		}
	}

	comp := compiler.New()
	err := comp.Compile(program)
	util.ErrorCheck(err)

	f.WriteConstant(comp.Bytecode().Constants)
	f.WriteInstruction(comp.Bytecode().Instructions, comp.Bytecode().Constants)

	c := f.ReadConstant()
	fmt.Println([]byte{c[6], c[12], c[18], c[24], c[25], c[26]})
}

/*
	scanner := bufio.NewScanner(os.Stdin)

	for {
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		l := lexer.New(line)
		p := parser.New(l)

		program := p.ParseProgram()
		if len(p.Errors()) != 0 {
			for _, err := range p.Errors() {
				io.WriteString(os.Stdout, err)
			}
			continue
		}
		io.WriteString(os.Stdout, program.String()+"\n")

		comp := compiler.New()
		err := comp.Compile(program)
		PrintComileByteCode(comp.Bytecode())
		if err != nil {
			fmt.Fprintf(os.Stdout, "Compile failed:\n %s\n\n", err)
			continue
		}

		machine := vm.New(comp.Bytecode())
		err = machine.Run()
		if err != nil {
			fmt.Fprintf(os.Stdout, "Executiong bytecode failed:\n %s\n\n", err)
			continue
		}

		stackTop := machine.LastPoppedStackElem()
		io.WriteString(os.Stdout, stackTop.Inspect())
		io.WriteString(os.Stdout, "\n\n")
	}
}

func PrintComileByteCode(compiler *compiler.Bytecode) {
	for ip := 0; ip < len(compiler.Instructions); ip++ {
		op := code.Opcode(compiler.Instructions[ip])

		switch op {
		case code.OpConstant:
			fmt.Printf("OpConstant : %d\n", code.ReadUint32(compiler.Instructions[ip+1:]))
			ip += 4
		case code.OpAdd:
			fmt.Printf("OpAdd\n")
		case code.OpSub:
			fmt.Printf("OpSub\n")
		case code.OpMul:
			fmt.Printf("OpMul\n")
		case code.OpDiv:
			fmt.Printf("OpDiv\n")

		}
	}
}

*/
