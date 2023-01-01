package scanner

import (
	"fmt"

	"github.com/HereIsKevin/edible/internal/logger"
)

type TokenKind uint8

const (
	// Controls
	TokenColon TokenKind = iota
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

	// End of File
	TokenEOF
)

func (kind TokenKind) String() string {
	switch kind {
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
	case TokenEOF:
		return "EOF"
	default:
		return "Unknown"
	}
}

type Token struct {
	Kind   TokenKind
	Lexeme string
	Span   logger.Span
}

func (token Token) String() string {
	switch token.Kind {
	case TokenStr, TokenIdent:
		return fmt.Sprint(token.Kind, "(\"", token.Lexeme, "\")")
	case TokenInt, TokenFloat:
		return fmt.Sprint(token.Kind, "(", token.Lexeme, ")")
	default:
		return token.Kind.String()
	}
}
