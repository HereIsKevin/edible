package parser

import (
	"strconv"

	"github.com/HereIsKevin/edible/internal/logger"
	"github.com/HereIsKevin/edible/internal/scanner"
)

type Parser struct {
	tokens  scanner.Tokens
	logger  *logger.Logger
	current int
}

func New(tokens scanner.Tokens, logger *logger.Logger) *Parser {
	return &Parser{
		tokens:  tokens,
		logger:  logger,
		current: 0,
	}
}

func (parser *Parser) Parse() Expr {
	expr := parser.parseBlock()

	if !parser.isEOF() {
		parser.addError("Unexpected token.", parser.peek())
	}

	return expr
}

func (parser *Parser) parseBlock() Expr {
	switch parser.peek().Kind {
	// Block expression
	case scanner.TokenOpenBlock:
		// Consume block open.
		parser.advance()

		// Consume expression and possibly exit fatally.
		expr := parser.parseBlock()
		if expr == nil {
			return nil
		}

		// Consume block closing.
		token := parser.consume(scanner.TokenCloseBlock, "Expect dedent after block")
		if token == nil {
			return nil
		}

		return expr

	// Block array
	case scanner.TokenDash:
		return parser.parseBlockArray()

	// Block table
	case scanner.TokenStr, scanner.TokenIdent:
		if parser.peekNext().Kind == scanner.TokenColon ||
			parser.peekNext().Kind == scanner.TokenLess {

			// Only parse as block table if the key is followed by a colon or inheritance
			// operator. Otherwise it must be an inline expression.
			return parser.parseBlockTable()
		}

		// Go on to inline expression otherwise.
		fallthrough

	default:
		// Descend to inline expressions.
		return parser.parseInline()
	}
}

func (parser *Parser) parseBlockArray() Expr {
	items := []Expr{}

	// Use position of first dash as start.
	openPos := parser.peek().Pos

	for parser.peek().Kind == scanner.TokenDash {
		// Consume dash.
		parser.advance()

		// Consume expression.
		expr := parser.parseBlock()
		if expr == nil {
			return nil
		}

		// Add expression as item.
		items = append(items, expr)

		// Finished if there is no newline.
		if parser.peek().Kind != scanner.TokenNewline {
			break
		}

		// Consume newline and repeat if dash.
		parser.advance()
	}

	// Use position of last token as end.
	closePos := parser.previous().Pos

	return &ExprArray{
		Items: items,
		Position: logger.Pos{
			Start: openPos.Start,
			End:   closePos.End,
			Line:  openPos.Line,
		},
	}
}

func (parser *Parser) parseBlockTable() Expr {
	items := []*TableItem{}

	// Use position of first key as start.
	openPos := parser.peek().Pos

	for parser.peek().Kind == scanner.TokenStr ||
		parser.peek().Kind == scanner.TokenIdent {

		// Consume table item.
		item := parser.parseTableItem(parser.parseBlock)
		if item == nil {
			return nil
		}

		// Add item to table.
		items = append(items, item)

		// Finished if there is no newline.
		if parser.peek().Kind != scanner.TokenNewline {
			break
		}

		// Consume newline and repeat if key.
		parser.advance()
	}

	// Use position of last token as end.
	closePos := parser.previous().Pos

	return &ExprTable{
		Items: items,
		Position: logger.Pos{
			Start: openPos.Start,
			End:   closePos.End,
			Line:  openPos.Line,
		},
	}
}

func (parser *Parser) parseInline() Expr {
	return parser.parseTerm()
}

func (parser *Parser) parseTerm() Expr {
	expr := parser.parseFactor()
	if expr == nil {
		return nil
	}

loop:
	for {
		var op BinaryOp

		switch parser.peek().Kind {
		case scanner.TokenPlus:
			op = BinaryPlus
		case scanner.TokenMinus:
			op = BinaryMinus
		default:
			break loop
		}

		pos := parser.advance().Pos
		right := parser.parseFactor()
		if right == nil {
			return nil
		}

		expr = &ExprBinary{
			Left:     expr,
			Op:       op,
			Right:    right,
			Position: pos,
		}
	}

	return expr
}

func (parser *Parser) parseFactor() Expr {
	expr := parser.parseUnary()
	if expr == nil {
		return nil
	}

loop:
	for {
		var op BinaryOp

		switch parser.peek().Kind {
		case scanner.TokenStar:
			op = BinaryStar
		case scanner.TokenSlash:
			op = BinarySlash
		default:
			break loop
		}

		pos := parser.advance().Pos
		right := parser.parseUnary()
		if right == nil {
			return nil
		}

		expr = &ExprBinary{
			Left:     expr,
			Op:       op,
			Right:    right,
			Position: pos,
		}
	}

	return expr
}

func (parser *Parser) parseUnary() Expr {
	var op UnaryOp

	switch parser.peek().Kind {
	case scanner.TokenPlus:
		op = UnaryPlus
	case scanner.TokenMinus:
		op = UnaryMinus
	default:
		return parser.parseLiteral()
	}

	pos := parser.advance().Pos
	expr := parser.parseUnary()
	if expr == nil {
		return nil
	}

	return &ExprUnary{
		Op:       op,
		Right:    expr,
		Position: pos,
	}
}

func (parser *Parser) parseLiteral() Expr {
	switch parser.peek().Kind {
	// String
	case scanner.TokenStr:
		token := parser.advance()

		return &ExprStr{
			Value:    token.Value,
			Position: token.Pos,
		}

	// Identifier, should only be keywords
	case scanner.TokenIdent:
		token := parser.advance()

		switch token.Value {
		case "true":
			return &ExprBool{
				Value:    true,
				Position: token.Pos,
			}

		case "false":
			return &ExprBool{
				Value:    false,
				Position: token.Pos,
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
			Value:    value,
			Position: token.Pos,
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
			Value:    value,
			Position: token.Pos,
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
		token := parser.consume(scanner.TokenCloseParen, "Expect ')' after expression")
		if token == nil {
			return nil
		}

		return expr

	// Reference
	case scanner.TokenDollar, scanner.TokenDot:
		return parser.parseRef()

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

func (parser *Parser) parseRef() Expr {
	keys := []Expr{}

	// Consume modifier.
	modifierToken := parser.advance()
	modifierPos := modifierToken.Pos
	modifier := RefRelative

	// Change to absolute refernce if there is an absolute modifier.
	if modifierToken.Kind == scanner.TokenDollar {
		modifier = RefAbsolute
	}

	// Take identifier as key if possible.
	if parser.peek().Kind == scanner.TokenIdent {
		token := parser.advance()
		keys = append(keys, &ExprStr{
			Value:    token.Value,
			Position: token.Pos,
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
				Value:    token.Value,
				Position: token.Pos,
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
		Modifier: modifier,
		Keys:     keys,
		Position: logger.Pos{
			Start: modifierPos.Start,
			End:   parser.previous().Pos.End,
			Line:  modifierPos.Line,
		},
	}
}

func (parser *Parser) parseInlineArray() Expr {
	items := []Expr{}

	// Consume opening bracket and take position.
	openPos := parser.advance().Pos

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
	token := parser.consume(scanner.TokenCloseBrack, "Expect ']' after array.")
	if token == nil {
		return nil
	}

	// Take position from closing bracket.
	closePos := token.Pos

	return &ExprArray{
		Items: items,
		Position: logger.Pos{
			Start: openPos.Start,
			End:   closePos.End,
			Line:  openPos.Line,
		},
	}
}

func (parser *Parser) parseInlineTable() Expr {
	items := []*TableItem{}

	// Consume opening brace.
	openPos := parser.advance().Pos

	for parser.peek().Kind != scanner.TokenCloseBrace {
		// Consume table item.
		item := parser.parseTableItem(parser.parseInline)
		if item == nil {
			return nil
		}

		// Add item to table.
		items = append(items, item)

		// Check for comma if not at closing brace, otherwise just repeat.
		if parser.peek().Kind != scanner.TokenCloseBrace {
			// Consume comma.
			token := parser.consume(scanner.TokenComma, "Expect ',' between items.")
			if token == nil {
				return nil
			}
		}
	}

	// Consume closing brace.
	token := parser.consume(scanner.TokenCloseBrace, "Expect '}' after table.")
	if token == nil {
		return nil
	}

	// Take position from closing brace.
	closePos := token.Pos

	return &ExprTable{
		Items: items,
		Position: logger.Pos{
			Start: openPos.Start,
			End:   closePos.End,
			Line:  openPos.Line,
		},
	}
}

func (parser *Parser) parseTableItem(valueParser func() Expr) *TableItem {
	var key Expr

	if parser.peek().Kind == scanner.TokenStr ||
		parser.peek().Kind == scanner.TokenIdent {

		// Consume the key token.
		token := parser.advance()

		// Create string expression for key.
		key = &ExprStr{
			Value:    token.Value,
			Position: token.Pos,
		}
	} else {
		// Fatal, cannot recover from missing key.
		parser.addError("Expect string or identifier for key.", parser.peek())
		return nil
	}

	var parent Expr

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
	value := valueParser()
	if value == nil {
		return nil
	}

	return &TableItem{
		Key:    key,
		Parent: parent,
		Value:  value,
	}
}

func (parser *Parser) isEOF() bool {
	return parser.peek().Kind == scanner.TokenEOF
}

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

func (parser *Parser) peekNext() *scanner.Token {
	if len(parser.tokens) > parser.current+1 {
		return &parser.tokens[parser.current+1]
	}

	return &parser.tokens[len(parser.tokens)-1]
}

func (parser *Parser) addError(message string, token *scanner.Token) {
	parser.logger.Add(message, token.Pos)
}
