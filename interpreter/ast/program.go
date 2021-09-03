package ast

import (
	"bytes"
	"fmt"
)

type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	} else {
		return ""
	}
}

func (p *Program) String() string {
	var out bytes.Buffer

	for _, s := range p.Statements {
		fmt.Println(s)
		out.WriteString(s.String())
	}
	return out.String()
}
