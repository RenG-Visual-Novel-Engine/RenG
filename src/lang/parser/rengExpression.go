package parser

import (
	"RenG/src/config"
	"RenG/src/lang/ast"
	"RenG/src/lang/token"
	"strconv"
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

func (p *Parser) parseMenuExpression() ast.Expression {
	exp := &ast.MenuExpression{Token: p.curToken}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	p.expectPeek(token.ENDSENTENCE)
	p.nextToken()

	for p.curTokenIs(token.STRING) {
		exp.Key = append(exp.Key, p.parseExpression(LOWEST))
		if !p.expectPeek(token.LBRACE) {
			return nil
		}

		exp.Action = append(exp.Action, p.parseBlockStatement())
		p.expectPeek(token.ENDSENTENCE)
		p.nextToken()
	}

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

func (p *Parser) parseTextExpression() ast.Expression {
	exp := &ast.TextExpression{Token: p.curToken}

	p.nextToken()

	exp.Text = p.parseExpression(LOWEST)
	exp.Transform = &ast.Identifier{
		Token: token.Token{
			Type:    token.IDENT,
			Literal: "IDENT",
		},
		Value: "default",
	}
	exp.Style = &ast.Identifier{
		Token: token.Token{
			Type:    token.IDENT,
			Literal: "IDENT",
		},
		Value: "defaultStyle",
	}
	exp.Width = &ast.IntegerLiteral{
		Token: token.Token{
			Type:    token.INT,
			Literal: strconv.Itoa(config.Width),
		},
		Value: int64(config.Width),
	}
	exp.Typing = &ast.Boolean{
		Token: token.Token{
			Type:    token.FALSE,
			Literal: "false",
		},
		Value: false,
	}

	for p.expectPeek(token.AT) ||
		p.expectPeek(token.AS) ||
		p.expectPeek(token.LIMITWIDTH) ||
		p.expectPeek(token.TYPINGEFFECT) {
		switch p.curToken.Type {
		case token.AT:
			p.nextToken()
			exp.Transform = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		case token.AS:
			p.nextToken()
			exp.Style = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		case token.LIMITWIDTH:
			p.nextToken()
			exp.Width = p.parseExpression(LOWEST)
		case token.TYPINGEFFECT:
			p.nextToken()
			exp.Typing = p.parseExpression(LOWEST)
		}
	}

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
		switch p.curToken.Type {
		case token.AT:
			p.nextToken()
			exp.Transform = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		case token.ACTION:
			p.nextToken()
			exp.Action = p.parseExpression(LOWEST)
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
	exp.Style = &ast.Identifier{
		Token: token.Token{
			Type:    token.IDENT,
			Literal: "IDENT",
		},
		Value: "defaultStyle",
	}

	for p.expectPeek(token.AT) ||
		p.expectPeek(token.AS) ||
		p.expectPeek(token.ACTION) {
		switch p.curToken.Type {
		case token.AT:
			p.nextToken()
			exp.Transform = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		case token.AS:
			p.nextToken()
			exp.Style = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		case token.ACTION:
			p.nextToken()
			exp.Action = p.parseExpression(LOWEST)
		}
	}

	return exp
}

func (p *Parser) parseKeyExpression() ast.Expression {
	exp := &ast.KeyExpression{Token: p.curToken}

	p.nextToken()

	exp.Key = p.parseExpression(LOWEST)

	if !p.expectPeek(token.ACTION) {
		return nil
	}

	p.nextToken()

	exp.Action = p.parseExpression(LOWEST)

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

func (p *Parser) parseStyleExpression() ast.Expression {
	exp := &ast.StyleExpression{Token: p.curToken}

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

	exp.Value = p.parseExpression(LOWEST)

	return exp
}

func (p *Parser) parseYposExpression() ast.Expression {
	exp := &ast.YPosExpression{Token: p.curToken}

	p.nextToken()

	exp.Value = p.parseExpression(LOWEST)

	return exp
}

func (p *Parser) parseXSizeExpression() ast.Expression {
	exp := &ast.XSizeExpression{Token: p.curToken}

	p.nextToken()

	exp.Value = p.parseExpression(LOWEST)

	return exp
}

func (p *Parser) parseYSizeExpression() ast.Expression {
	exp := &ast.YSizeExpression{Token: p.curToken}

	p.nextToken()

	exp.Value = p.parseExpression(LOWEST)

	return exp
}

func (p *Parser) parseRotateExpression() ast.Expression {
	exp := &ast.RotateExpression{Token: p.curToken}

	p.nextToken()

	exp.Value = p.parseExpression(LOWEST)

	return exp
}

func (p *Parser) parseAlphaExpression() ast.Expression {
	exp := &ast.AlphaExpression{Token: p.curToken}

	p.nextToken()

	exp.Value = p.parseExpression(LOWEST)

	return exp
}

func (p *Parser) parseColorExpression() ast.Expression {
	exp := &ast.ColorExpression{Token: p.curToken}

	p.nextToken()

	exp.Value = p.parseExpression(LOWEST)

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

func (p *Parser) parseWhoExpression() ast.Expression {
	return &ast.WhoExpression{Token: p.curToken}
}

func (p *Parser) parseWhatExpression() ast.Expression {
	return &ast.WhatExpression{Token: p.curToken}
}

func (p *Parser) parseItemsExpression() ast.Expression {
	return &ast.ItemsExpression{Token: p.curToken}
}
