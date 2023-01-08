package scanner

import (
	"fmt"
	"strings"

	"github.com/HereIsKevin/edible/internal/logger"
)

type TokenKind uint8

const (
	// End of File
	TokenEOF TokenKind = iota

	// Controls
	TokenColon
	TokenComma
	TokenDash

	// Operators
	TokenPlus
	TokenMinus
	TokenStar
	TokenSlash
	TokenLess
	TokenDot
	TokenDollar

	// Delimiters
	TokenOpenParen
	TokenCloseParen
	TokenOpenBrack
	TokenCloseBrack
	TokenOpenBrace
	TokenCloseBrace

	// Whitespace
	TokenNewline
	TokenOpenBlock
	TokenCloseBlock

	// Literals
	TokenStr
	TokenIdent
	TokenInt
	TokenFloat
)

func (kind TokenKind) String() string {
	switch kind {
	case TokenEOF:
		return "EOF"
	case TokenColon:
		return "Colon"
	case TokenComma:
		return "Comma"
	case TokenDash:
		return "Dash"
	case TokenPlus:
		return "Plus"
	case TokenMinus:
		return "Minus"
	case TokenStar:
		return "Star"
	case TokenSlash:
		return "Slash"
	case TokenLess:
		return "Less"
	case TokenDot:
		return "Dot"
	case TokenDollar:
		return "Dollar"
	case TokenOpenParen:
		return "OpenParen"
	case TokenCloseParen:
		return "CloseParen"
	case TokenOpenBrack:
		return "OpenBrack"
	case TokenCloseBrack:
		return "CloseBrack"
	case TokenOpenBrace:
		return "OpenBrace"
	case TokenCloseBrace:
		return "CloseBrace"
	case TokenNewline:
		return "Newline"
	case TokenOpenBlock:
		return "OpenBlock"
	case TokenCloseBlock:
		return "CloseBlock"
	case TokenStr:
		return "Str"
	case TokenIdent:
		return "Ident"
	case TokenInt:
		return "Int"
	case TokenFloat:
		return "Float"
	default:
		return "Unknown"
	}
}

type Token struct {
	Kind  TokenKind
	Value string
	Pos   logger.Pos
}

func (token Token) String() string {
	switch token.Kind {
	case TokenStr:
		return fmt.Sprintf("%s(\"%s\")", token.Kind, token.Value)
	case TokenIdent, TokenInt, TokenFloat:
		return fmt.Sprintf("%s(%s)", token.Kind, token.Value)
	default:
		return token.Kind.String()
	}
}

type Tokens []Token

func (tokens Tokens) String() string {
	depth := 0
	builder := strings.Builder{}

	for index, token := range tokens {
		switch token.Kind {
		case TokenOpenParen, TokenOpenBrack, TokenOpenBrace, TokenOpenBlock:
			depth += 1
			builder.WriteString(fmt.Sprintf("%s\n%s", token, indent(depth)))
		case TokenCloseParen, TokenCloseBrack, TokenCloseBrace, TokenCloseBlock:
			depth -= 1
			builder.WriteString(fmt.Sprintf("\n%s%s ", indent(depth), token))
		case TokenEOF, TokenComma, TokenNewline:
			if len(tokens) > index+1 {
				kind := tokens[index+1].Kind

				if kind == TokenCloseParen ||
					kind == TokenCloseBrack ||
					kind == TokenCloseBrace ||
					kind == TokenCloseBlock {

					builder.WriteString(token.String())
					break
				}
			}

			builder.WriteString(fmt.Sprintf("%s\n%s", token, indent(depth)))
		default:
			builder.WriteString(fmt.Sprintf("%s ", token.String()))
		}
	}

	return strings.TrimSpace(builder.String())
}

func indent(depth int) string {
	return strings.Repeat("    ", depth)
}
