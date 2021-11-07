package parser

import (
	"RenG/src/lang/ast"
	"RenG/src/lang/token"
)

func (p *Parser) parseScreenExpression() ast.Expression {
	exp := &ast.ScreenExpression{Token: p.curToken}

	p.nextToken()

	exp.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	exp.Body = p.parseBlockStatement()

	return exp
}

func (p *Parser) parseLabelExpression() ast.Expression {
	exp := &ast.LabelExpression{Token: p.curToken}

	p.nextToken()

	exp.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	exp.Body = p.parseBlockStatement()

	return exp
}

func (p *Parser) parseCallLabelExpression() ast.Expression {
	exp := &ast.CallLabelExpression{Token: p.curToken}

	p.nextToken()

	exp.Label = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	return exp
}

func (p *Parser) parseJumpLabelExpression() ast.Expression {
	exp := &ast.JumpLabelExpression{Token: p.curToken}

	p.nextToken()

	exp.Label = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	return exp
}

func (p *Parser) parseImagebuttonExpression() ast.Expression {
	exp := &ast.ImagebuttonExpression{Token: p.curToken}

	p.nextToken()

	exp.MainImage = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	exp.Transform = &ast.Identifier{
		Token: token.Token{
			Type:    token.IDENT,
			Literal: "IDENT",
		},
		Value: "default",
	}

	for p.expectPeek(token.AT) || p.expectPeek(token.ACTION) {
		if p.curTokenIs(token.ACTION) {
			p.nextToken()

			exp.Action = p.parseExpression(LOWEST)
		} else if p.curTokenIs(token.AT) {
			p.nextToken()

			exp.Transform = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		}
	}

	return exp
}

func (p *Parser) parseTextbuttonExpression() ast.Expression {
	exp := &ast.TextbuttonExpression{Token: p.curToken}

	p.nextToken()

	exp.Text = p.parseExpression(LOWEST)
	exp.Transform = &ast.Identifier{
		Token: token.Token{
			Type:    token.IDENT,
			Literal: "IDENT",
		},
		Value: "default",
	}

	for p.expectPeek(token.AT) || p.expectPeek(token.ACTION) {
		if p.curTokenIs(token.ACTION) {
			p.nextToken()

			exp.Action = p.parseExpression(LOWEST)
		} else if p.curTokenIs(token.AT) {
			p.nextToken()

			exp.Transform = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		}
	}

	return exp
}

func (p *Parser) parseImageExpression() ast.Expression {
	exp := &ast.ImageExpression{Token: p.curToken}

	p.nextToken()

	exp.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()

	exp.Path = p.parseExpression(LOWEST)

	return exp
}

func (p *Parser) parseVideoExpression() ast.Expression {
	exp := &ast.VideoExpression{Token: p.curToken}
	info := make(map[string]ast.Expression)

	p.nextToken()

	exp.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	var arg string

	p.nextToken()

	arg = p.curToken.Literal

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()

	info[arg] = p.parseExpression(LOWEST)

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()

		arg = p.curToken.Literal

		if !p.expectPeek(token.ASSIGN) {
			return nil
		}

		p.nextToken()

		info[arg] = p.parseExpression(LOWEST)
	}

	exp.Info = info

	return exp
}

func (p *Parser) parseShowExpression() ast.Expression {
	exp := &ast.ShowExpression{Token: p.curToken}

	p.nextToken()

	exp.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if p.peekTokenIs(token.AT) {
		p.nextToken()
		p.nextToken()
		exp.Transform = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	} else {
		exp.Transform = &ast.Identifier{
			Token: token.Token{
				Type:    token.IDENT,
				Literal: "IDENT",
			},
			Value: "default",
		}
	}

	return exp
}

func (p *Parser) parseHideExpression() ast.Expression {
	exp := &ast.HideExpression{Token: p.curToken}

	p.nextToken()

	exp.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	return exp
}

func (p *Parser) parseTranformExpression() ast.Expression {
	exp := &ast.TransformExpression{Token: p.curToken}

	p.nextToken()

	exp.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	exp.Body = p.parseBlockStatement()

	return exp
}

func (p *Parser) parseXposExpression() ast.Expression {
	exp := &ast.XPosExpression{Token: p.curToken}

	p.nextToken()

	exp.Value = p.parseExpression(PREFIX)

	return exp
}

func (p *Parser) parseYposExpression() ast.Expression {
	exp := &ast.YPosExpression{Token: p.curToken}

	p.nextToken()

	exp.Value = p.parseExpression(PREFIX)

	return exp
}

func (p *Parser) parsePlayExpression() ast.Expression {
	exp := &ast.PlayExpression{Token: p.curToken}

	p.nextToken()

	exp.Channel = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if p.expectPeek(token.IDENT) {
		exp.Loop = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	} else {

		switch exp.Channel.Value {
		case "music":
			exp.Loop = &ast.Identifier{
				Token: token.Token{
					Type:    token.IDENT,
					Literal: "IDENT",
				},
				Value: "loop",
			}
		default:
			exp.Loop = &ast.Identifier{
				Token: token.Token{
					Type:    token.IDENT,
					Literal: "IDENT",
				},
				Value: "noloop",
			}
		}
	}

	p.nextToken()

	exp.Music = p.parseExpression(LOWEST)

	return exp
}

func (p *Parser) parseStopExpression() ast.Expression {
	exp := &ast.StopExpression{Token: p.curToken}

	p.nextToken()

	exp.Channel = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	return exp
}
