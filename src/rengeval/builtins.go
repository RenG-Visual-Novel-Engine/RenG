package rengeval

import (
	"RenG/src/ast"
	"RenG/src/token"
)

var screenBuiltins = map[string]*ast.TransformExpression{
	"default": {
		Token: token.Token{
			Type:    token.TRANSFORM,
			Literal: "transform",
		},
		Name: &ast.Identifier{
			Token: token.Token{
				Type:    token.IDENT,
				Literal: "IDENT",
			},
			Value: "default",
		},
		Body: &ast.BlockStatement{
			Token: token.Token{
				Type:    token.LBRACE,
				Literal: "{",
			},
			Statements: []ast.Statement{},
		},
	},
}
