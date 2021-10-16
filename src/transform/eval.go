package transform

import (
	sdl "RenG/src/SDL"
	"RenG/src/ast"
	"RenG/src/config"
	"RenG/src/evaluator"
	"RenG/src/object"
	"RenG/src/token"
	"strconv"
)

// TODO : 연산자 표현식 평가하기
func TransformEval(node ast.Node, texture *sdl.SDL_Texture, env *object.Environment) object.Object {
	switch node := node.(type) {
	case *ast.BlockStatement:
		return evalRengBlockStatements(node, texture, env)
	case *ast.ExpressionStatement:
		return TransformEval(node.Expression, texture, env)
	case *ast.TransformExpression:
		return evalTransformExpression(node, texture, env)
	case *ast.XPosExpression:
		result := evaluator.Eval(node.Value, env)
		xpos := result.(*object.Integer).Value
		texture.Xpos = int(xpos)
	case *ast.YPosExpression:
		result := evaluator.Eval(node.Value, env)
		ypos := result.(*object.Integer).Value
		texture.Ypos = int(ypos)
	}
	return nil
}

func evalRengBlockStatements(block *ast.BlockStatement, texture *sdl.SDL_Texture, env *object.Environment) object.Object {
	var result object.Object

	for _, statement := range block.Statements {
		result = TransformEval(statement, texture, env)
		if result != nil {
			rt := result.Type()
			if rt == object.RETURN_VALUE_OBJ || rt == object.ERROR_OBJ {
				return result
			}
		}
	}

	return result
}

func evalTransformExpression(transform *ast.TransformExpression, texture *sdl.SDL_Texture, env *object.Environment) object.Object {
	switch transform.Name.Value {
	case "default":
		transform.Body.Statements = append(transform.Body.Statements, &ast.ExpressionStatement{
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
								Literal: strconv.Itoa(texture.Width),
							},
							Value: int64(texture.Width),
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
		})
		transform.Body.Statements = append(transform.Body.Statements, &ast.ExpressionStatement{
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
								Literal: strconv.Itoa(texture.Height),
							},
							Value: int64(texture.Height),
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
		})
	}
	TransformEval(transform.Body, texture, env)

	return nil
}
