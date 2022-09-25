package main

import (
	"RenG/Compiler/compiler"
	"RenG/Compiler/lexer"
	"RenG/Compiler/parser"
	"RenG/Compiler/vm"
	"bufio"
	"fmt"
	"io"
	"os"
)

func main() {
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

		stackTop := machine.StackTop()
		io.WriteString(os.Stdout, stackTop.Inspect())
		io.WriteString(os.Stdout, "\n\n")
	}
}
