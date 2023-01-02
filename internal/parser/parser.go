package parser

import (
	"strconv"

	"github.com/HereIsKevin/edible/internal/logger"
	"github.com/HereIsKevin/edible/internal/scanner"
)

type Parser struct {
	tokens  []scanner.Token
	logger  *logger.Logger
	current int
}

func New(tokens []scanner.Token, logger *logger.Logger) *Parser {
	return &Parser{
		tokens:  tokens,
		logger:  logger,
		current: 0,
	}
}

func (parser *Parser) Parse() Expr {
	expr := parser.parseLiteral()

	if !parser.isEOF() {
		parser.addError("Unexpected token.", parser.peek())
	}

	return expr
}

func (parser *Parser) parseInline() Expr {
	return parser.parseLiteral()
}

func (parser *Parser) parseLiteral() Expr {
	switch parser.peek().Kind {
	// String
	case scanner.TokenStr:
		token := parser.advance()

		return &ExprStr{
			Value:     token.Value,
			ValueSpan: token.Span,
		}

	// Identifier, should only be keywords
	case scanner.TokenIdent:
		token := parser.advance()

		switch token.Value {
		case "true":
			return &ExprBool{
				Value:     true,
				ValueSpan: token.Span,
			}

		case "false":
			return &ExprBool{
				Value:     false,
				ValueSpan: token.Span,
			}

		default:
			// Fatal, cannot recover from random identifiers that are not keywords.
			parser.addError("Unexpected identifier.", parser.previous())
			return nil
		}

	// Integer
	case scanner.TokenInt:
		token := parser.advance()
		value, err := strconv.ParseInt(token.Value, 10, 64)
		if err != nil {
			// Recover by assuming it is 0.
			parser.addError("Integer out of range.", parser.previous())
		}

		return &ExprInt{
			Value:     value,
			ValueSpan: token.Span,
		}

	// Float
	case scanner.TokenFloat:
		token := parser.advance()
		value, err := strconv.ParseFloat(token.Value, 64)
		if err != nil {
			// Recover by assuming it is 0.
			parser.addError("Float out of range.", parser.previous())
		}

		return &ExprFloat{
			Value:     value,
			ValueSpan: token.Span,
		}

	// Grouping
	case scanner.TokenOpenParen:
		// Consume open parenthesis.
		parser.advance()

		// Consume expression and possibly exit fatally.
		expr := parser.parseInline()
		if expr == nil {
			return nil
		}

		// Consume closing parenthesis.
		token := parser.consume(scanner.TokenCloseBrack, "Expect ')' after parenthesis")
		if token == nil {
			return nil
		}

		return expr

	// Reference
	case scanner.TokenDollar, scanner.TokenDot:
		return parser.parseReference()

	// Inline array
	case scanner.TokenOpenBrack:
		return parser.parseInlineArray()

	// Inline table
	case scanner.TokenOpenBrace:
		return parser.parseInlineTable()

	default:
		// Fatal, cannot recover from random tokens.
		parser.addError("Expect literal.", parser.previous())
		return nil
	}
}

func (parser *Parser) parseReference() Expr {
	keys := []Expr{}

	// Consume modifier.
	modifierToken := parser.advance()
	modifierSpan := modifierToken.Span
	modifier := RefRelative

	// Change to absolute refernce if there is an absolute modifier.
	if modifierToken.Kind == scanner.TokenDollar {
		modifier = RefAbsolute
	}

	// Take identifier as key if possible.
	if parser.peek().Kind == scanner.TokenIdent {
		token := parser.advance()
		keys = append(keys, &ExprStr{
			Value:     token.Value,
			ValueSpan: token.Span,
		})
	}

loop:
	for {
		switch parser.peek().Kind {
		// Literal key
		case scanner.TokenDot:
			// Do not accept extra dots after modifiers.
			if len(keys) == 0 {
				// Recover by ignoring dot.
				parser.addError("Unnecessary '.' after '$' or '.'.", parser.peek())
			}

			// Consume dot.
			parser.advance()

			// Consume identifier as key.
			token := parser.consume(scanner.TokenIdent, "Expect identifier key.")
			if token == nil {
				return nil
			}

			keys = append(keys, &ExprStr{
				Value:     token.Value,
				ValueSpan: token.Span,
			})

		// Expression key
		case scanner.TokenOpenBrack:
			// Consume opening bracket.
			parser.advance()

			// Consume expression and exit if fatal.
			expr := parser.parseInline()
			if expr == nil {
				return nil
			}

			// Add expression as key
			keys = append(keys, expr)

			// Consume closing bracket.
			token := parser.consume(scanner.TokenCloseBrack, "Expect ']' after expresion key.")
			if token == nil {
				return nil
			}

		default:
			break loop
		}
	}

	return &ExprRef{
		Modifier:     modifier,
		ModifierSpan: modifierSpan,
		Keys:         keys,
	}
}

func (parser *Parser) parseInlineArray() Expr {
	items := []Expr{}

	// Consume opening bracket and take span.
	openSpan := parser.advance().Span

	for parser.peek().Kind != scanner.TokenCloseBrack {
		// Consume expression.
		expr := parser.parseInline()
		if expr == nil {
			return nil
		}

		// Add expression as item.
		items = append(items, expr)

		// Check for comma if not at closing bracket, otherwise just repeat.
		if parser.peek().Kind != scanner.TokenCloseBrack {
			// Consume comma.
			token := parser.consume(scanner.TokenComma, "Expect ',' between items.")
			if token == nil {
				return nil
			}
		}
	}

	// Consume closing bracket.
	token := parser.consume(scanner.TokenCloseBrack, "Expect ']' after expression key.")
	if token == nil {
		return nil
	}

	// Take span from closing bracket.
	closeSpan := token.Span

	return &ExprArray{
		OpenSpan:  openSpan,
		Items:     items,
		CloseSpan: closeSpan,
	}
}

func (parser *Parser) parseInlineTable() Expr {
	items := []TableItem{}

	// Consume opening brace.
	openSpan := parser.advance().Span

	for parser.peek().Kind != scanner.TokenCloseBrace {
		key := Expr(nil)

		switch parser.peek().Kind {
		// Literal key
		case scanner.TokenStr, scanner.TokenIdent:
			token := parser.advance()
			key = &ExprStr{
				Value:     token.Value,
				ValueSpan: token.Span,
			}

		// Expression key
		case scanner.TokenOpenBrack:
			// Consume opening bracket.
			parser.advance()

			// Consume expression and exit if fatal.
			key = parser.parseInline()
			if key == nil {
				return nil
			}

			// Add expression as key

			// Consume closing bracket.
			token := parser.consume(scanner.TokenCloseBrack, "Expect ']' after expresion key.")
			if token == nil {
				return nil
			}

		default:
			// Fatal, cannot recover from missing key.
			parser.addError("Expect string, identifier, or expression for key.", parser.peek())
			return nil
		}

		parent := Expr(nil)

		// Check for parent and consume if found.
		if parser.peek().Kind == scanner.TokenLess {
			// Consume inheritance operator.
			parser.advance()

			// Consume parent expression.
			parent = parser.parseInline()
			if parent == nil {
				return nil
			}
		}

		// Consume colon separator.
		token := parser.consume(scanner.TokenColon, "Expect ':' beween key and value.")
		if token == nil {
			return nil
		}

		// Consume value expression.
		value := parser.parseInline()
		if value == nil {
			return nil
		}

		// Add key and value pair as item.
		items = append(items, TableItem{
			Key:      key,
			Inherits: parent,
			Value:    value,
		})

		// Check for comma if not at closing bracket, otherwise just repeat.
		if parser.peek().Kind != scanner.TokenCloseBrack {
			// Consume comma.
			token := parser.consume(scanner.TokenComma, "Expect ',' between items.")
			if token == nil {
				return nil
			}
		}
	}

	// Consume closing brace.
	closeSpan := parser.advance().Span

	return &ExprTable{
		OpenSpan:  openSpan,
		Items:     items,
		CloseSpan: closeSpan,
	}
}

func (parser *Parser) isEOF() bool {
	return parser.peek().Kind == scanner.TokenEOF
}

// func (parser *Parser) check(kind scanner.TokenKind) bool {
// 	if parser.isEOF() {
// 		return false
// 	}

// 	return parser.peek().Kind == kind
// }

// func (parser *Parser) match(kinds... scanner.TokenKind) bool {
// 	actualKind := parser.peek().Kind

// 	for _, kind := range kinds {
// 		if kind == actualKind {
// 			parser.advance()
// 			return true
// 		}
// 	}

// 	return false
// }

func (parser *Parser) consume(expected scanner.TokenKind, message string) *scanner.Token {
	if parser.peek().Kind == expected {
		return parser.advance()
	}

	parser.addError(message, parser.peek())
	return nil
}

func (parser *Parser) advance() *scanner.Token {
	if !parser.isEOF() {
		parser.current++
	}

	return parser.previous()
}

func (parser *Parser) previous() *scanner.Token {
	return &parser.tokens[parser.current-1]
}

func (parser *Parser) peek() *scanner.Token {
	return &parser.tokens[parser.current]
}

func (parser *Parser) addError(message string, token *scanner.Token) {
	parser.logger.Add(message, token.Span)
}
