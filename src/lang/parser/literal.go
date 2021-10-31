package parser

import (
	"RenG/src/lang/ast"
	"RenG/src/lang/token"
	"fmt"
	"strconv"
)

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as integer", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	lit := &ast.FloatLiteral{Token: p.curToken}

	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as float", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	result := &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}

	var i int = 0

	for p.peekTokenIs(token.LBRACKET) {

		p.nextToken()
		p.nextToken()

		result.Exp = append(result.Exp, ast.ExpressionIndex{Exp: p.parseExpression(LOWEST), Index: i})

		i++

		p.nextToken()

		if p.peekToken.Type == token.STRING {
			p.nextToken()
			result.Values = append(result.Values, ast.StringIndex{Str: p.curToken.Literal, Index: i})
			i++
		}
	}

	return result
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}

	array.Elements = p.parseExpressionList(token.RBRACKET)

	return array
}
