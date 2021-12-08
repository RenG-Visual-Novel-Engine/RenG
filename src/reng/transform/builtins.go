package transform

import (
	"RenG/src/config"
	"RenG/src/lang/ast"
	"RenG/src/lang/token"
	"strconv"
)

func BuiltInsTransform(name string, w, h int) *ast.TransformExpression {
	switch name {
	case "default":
		return &ast.TransformExpression{
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
				Statements: []ast.Statement{
					&ast.ExpressionStatement{
						Token: token.Token{
							Type:    token.XPOS,
							Literal: "xpos",
						},
						Expression: &ast.XPosExpression{
							Token: token.Token{
								Type:    token.XPOS,
								Literal: "xpos",
							},
							Value: &ast.InfixExpression{
								Token: token.Token{
									Type:    token.SLASH,
									Literal: "/",
								},
								Left: &ast.InfixExpression{
									Token: token.Token{
										Type:    token.MINUS,
										Literal: "-",
									},
									Left: &ast.IntegerLiteral{
										Token: token.Token{
											Type:    token.INT,
											Literal: strconv.Itoa(config.Width),
										},
										Value: int64(config.Width),
									},
									Operator: "-",
									Right: &ast.IntegerLiteral{
										Token: token.Token{
											Type:    token.INT,
											Literal: strconv.Itoa(w),
										},
										Value: int64(w),
									},
								},
								Operator: "/",
								Right: &ast.IntegerLiteral{
									Token: token.Token{
										Type:    token.INT,
										Literal: "2",
									},
									Value: 2,
								},
							},
						},
					},
					&ast.ExpressionStatement{
						Token: token.Token{
							Type:    token.YPOS,
							Literal: "ypos",
						},
						Expression: &ast.YPosExpression{
							Token: token.Token{
								Type:    token.YPOS,
								Literal: "ypos",
							},
							Value: &ast.InfixExpression{
								Token: token.Token{
									Type:    token.SLASH,
									Literal: "/",
								},
								Left: &ast.InfixExpression{
									Token: token.Token{
										Type:    token.MINUS,
										Literal: "-",
									},
									Left: &ast.IntegerLiteral{
										Token: token.Token{
											Type:    token.INT,
											Literal: strconv.Itoa(config.Height),
										},
										Value: int64(config.Height),
									},
									Operator: "-",
									Right: &ast.IntegerLiteral{
										Token: token.Token{
											Type:    token.INT,
											Literal: strconv.Itoa(h),
										},
										Value: int64(h),
									},
								},
								Operator: "/",
								Right: &ast.IntegerLiteral{
									Token: token.Token{
										Type:    token.INT,
										Literal: "2",
									},
									Value: 2,
								},
							},
						},
					},
				},
			},
		}
	}
	return nil
}
