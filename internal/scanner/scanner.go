package scanner

import (
	"unicode"
	"unicode/utf8"

	"github.com/HereIsKevin/edible/internal/logger"
)

// TODO: Ternary operator.
// TODO: Bitwise operators.
// TODO: Boolean operators.
// TODO: Floor division operator.
// TODO: Basic builtin functions and custom functions.
type Scanner struct {
	source string
	logger *logger.Logger
	tokens Tokens

	indents     []int
	sensitivity int
	isLineStart bool

	start   int
	current int
	line    int
}

func New(source string, logger *logger.Logger) *Scanner {
	return &Scanner{
		source: source,
		logger: logger,
		tokens: make(Tokens, 0, len(source)/2),

		indents:     []int{},
		sensitivity: 0,
		isLineStart: false,

		start:   0,
		current: 0,
		line:    1,
	}
}

func (scanner *Scanner) Scan() Tokens {
	for !scanner.isEOF() {
		scanner.start = scanner.current
		scanner.scan()
	}

	// Auto-close all blocks by adding a dedent for every indent.
	for range scanner.indents {
		scanner.addToken(TokenCloseBlock)
	}

	// Add final EOF token.
	scanner.addToken(TokenEOF)

	return scanner.tokens
}

func (scanner *Scanner) scan() {
	character := scanner.advance()
	isLineStart := scanner.isLineStart
	scanner.isLineStart = false

	switch character {
	// Controls
	case ':':
		scanner.addToken(TokenColon)
	case ',':
		scanner.addToken(TokenComma)
	case '-':
		if isLineStart && scanner.isSensitive() {
			// Dashes at the start of a line in whitespace-sensitive areas denote arrays.
			scanner.scanDash()
		} else {
			// Otherwise, they are just a normal minus operator.
			scanner.addToken(TokenMinus)
		}

	// Operators
	case '+':
		scanner.addToken(TokenPlus)
	// See controls for minus operator.
	case '*':
		scanner.addToken(TokenStar)
	case '/':
		scanner.addToken(TokenSlash)
	case '<':
		scanner.addToken(TokenLess)
	case '.':
		scanner.addToken(TokenDot)
	case '$':
		scanner.addToken(TokenDollar)

	// Delimiters
	case '(':
		scanner.desensitize()
		scanner.addToken(TokenOpenParen)
	case ')':
		scanner.sensitize()
		scanner.addToken(TokenCloseParen)
	case '[':
		scanner.desensitize()
		scanner.addToken(TokenOpenBrack)
	case ']':
		scanner.sensitize()
		scanner.addToken(TokenCloseBrack)
	case '{':
		scanner.desensitize()
		scanner.addToken(TokenOpenBrace)
	case '}':
		scanner.sensitize()
		scanner.addToken(TokenCloseBrace)

	// Comments
	case '#':
		scanner.scanComment()

	// Whitespace: ' ', '\n', and '\r' ('\t' doesn't count)
	// Also known as the Morgoth, Sauron, and the Witch-King of Angmar

	//                     THE LORD OF THE RINGS
	//     Three Rings for the Elven-kings under the sky,
	//         Seven for the Dwarf-lords in their halls of stone,
	//     Nine for Mortal Men doomed to die,
	//         One for the Dark Lord on his dark throne
	//     In the Land of Mordor where the Shadows lie.
	//         One Ring to rule them all, One Ring to find them,
	//         One Ring to bring them all, and in the darkness bind them,
	//     In the Land of Mordor where the Shadows lie.

	case ' ':
		if isLineStart && scanner.isSensitive() {
			// Handle leading indentation. This should only run for indentation at the
			// beginning of the file since all indentation after newlines is consumed with
			// the newline. All other spaces are insignificant.
			scanner.scanBlock(character)
		}

	case '\t':
		if scanner.isSensitive() {
			// Tabs are never valid as indentation, and are only ignored while not sensitive
			// to whitespace. This is because though they count as one level while scanning,
			// the same as spaces, text editors usually display them as 4 or 8 spaces.
			scanner.addError("Unexpected tab, only spaces are valid indentation.")
		}

	case '\r', '\n':
		if scanner.isSensitive() {
			// Handle LF and CRLF line endings while sensitive to whitespace.
			scanner.scanBlock(character)
		}

	// Strings
	case '"':
		scanner.scanString()

	default:
		if isAlphabetic(character) {
			// Identifiers
			scanner.scanIdentifier()
		} else if isDigit(character) {
			// Numbers (Integers and Floats)
			scanner.scanNumber()
		} else if unicode.IsSpace(character) {
			if scanner.isSensitive() {
				// Ignore all whitespace while not sensitive to whitespace. However, if
				// sensitive whitespace, random whitespace is invalid.
				scanner.addError("Unexpected whitespace.")
			}
		} else {
			// Everything else, including random symbols, are invalid.
			scanner.addError("Unexpected character.")
		}
	}
}

func (scanner *Scanner) scanDash() {
	// Since looking ahead to see if there is an indented block or a single literal
	// results in excess complexity, sections after dashes are always contained
	// within blocks.

	// Start from 1 to compensate for the dash.
	indent := 1

	// All spaces after the dash count as indentation.
	for scanner.peek() == ' ' {
		scanner.advance()
		indent++
	}

	// Add the last indent to the current one if possible.
	if len(scanner.indents) > 0 {
		indent += scanner.indents[len(scanner.indents)-1]
	}

	// Create a dash and immediately begin a new block.
	scanner.addToken(TokenDash)
	scanner.addToken(TokenOpenBlock)

	scanner.indents = append(scanner.indents, indent)
}

func (scanner *Scanner) scanComment() {
	for scanner.peek() != '\r' && scanner.peek() != '\n' && !scanner.isEOF() {
		// Ignore everything until a newline or the end.
		scanner.advance()
	}
}

func (scanner *Scanner) scanBlock(previous rune) {
	indent := 0
	newline := false

	if previous == ' ' {
		indent++
	}

	if previous == '\n' {
		newline = true
	}

loop:
	for {
		switch scanner.peek() {
		case ' ':
			scanner.advance()
			indent++
		case '\r':
			// Consume CR
			scanner.advance()

			if scanner.peek() != '\n' {
				// CR-only line endings are invalid while sensitive to whitespace. However,
				// instead of going crazy, just skip over it to recover.
				scanner.addError("Unexpected CR, line endings are CRLF and LF.")
				continue
			}

			// Proceed to LF for CRLF
			fallthrough
		case '\n':
			// Consume LF
			scanner.advance()

			// Process LF and reset indent
			indent = 0
			newline = true
			scanner.isLineStart = true
		case '#':
			scanner.scanComment()
		default:
			break loop
		}
	}

	// Ignore all trailing whitespace
	if scanner.isEOF() {
		return
	}

	lastIndent := 0

	if len(scanner.indents) > 0 {
		lastIndent = scanner.indents[len(scanner.indents)-1]
	}

	if indent > lastIndent {
		// Add a new block if indent level increased.
		scanner.indents = append(scanner.indents, indent)
		scanner.addToken(TokenOpenBlock)
	} else if indent < lastIndent {
		for len(scanner.indents) > 0 && indent < scanner.indents[len(scanner.indents)-1] {
			// Close blocks until indent is at the correct level.
			scanner.indents = scanner.indents[:len(scanner.indents)-1]
			scanner.addToken(TokenCloseBlock)
		}

		// Dedenting always involves a newline
		scanner.addToken(TokenNewline)
	} else if newline && len(scanner.tokens) != 0 {
		// Add a newline if needed unless it is a leading newline
		scanner.addToken(TokenNewline)
	}

	// Mark the current state as the start of the line since indentation only occurs
	// at the start of lines. This allows leading dashes to be handled properly as
	// block arrays.
	scanner.isLineStart = true
}

// TODO: Multiline strings.
// TODO: Literal strings.
// TODO: Escapes codes.
// TODO: String interpolation.
func (scanner *Scanner) scanString() {
	for scanner.peek() != '"' && !scanner.isEOF() {
		if scanner.peek() == '\n' {
			scanner.addError("Unexpected newline within string.")
			return
		}

		scanner.advance()
	}

	if scanner.isEOF() {
		scanner.addError("Unterminated string.")
		return
	}

	// Consume closing '"'.
	scanner.advance()

	// This is always valid since ASCII characters should only occupy one byte.
	value := scanner.source[scanner.start+1 : scanner.current-1]

	// Create string token.
	scanner.addLiteralToken(TokenStr, value)
}

func (scanner *Scanner) scanIdentifier() {
	for isAlphanumeric(scanner.peek()) {
		scanner.advance()
	}

	// Take value from source.
	value := scanner.source[scanner.start:scanner.current]

	// Create identifier token.
	scanner.addLiteralToken(TokenIdent, value)
}

// TODO: Disallow leading zeros.
// TODO: Underscores in numbers.
// TODO: Add exponents.
// TODO: Add infinity and NaN.
func (scanner *Scanner) scanNumber() {
	for isDigit(scanner.peek()) {
		scanner.advance()
	}

	if scanner.peek() == '.' && isDigit(scanner.peekNext()) {
		// Consume decimal point.
		scanner.advance()

		for isDigit(scanner.peek()) {
			scanner.advance()
		}

		// Take value from source.
		value := scanner.source[scanner.start:scanner.current]

		// Create float token.
		scanner.addLiteralToken(TokenFloat, value)
	} else {
		// Take value from source.
		value := scanner.source[scanner.start:scanner.current]

		// Create integer token.
		scanner.addLiteralToken(TokenInt, value)
	}
}

func (scanner *Scanner) isEOF() bool {
	return scanner.current >= len(scanner.source)
}

func (scanner *Scanner) advance() rune {
	codePoint, width := utf8.DecodeRuneInString(scanner.source[scanner.current:])
	if width == 0 {
		return '\u0000'
	}

	scanner.current += width

	if codePoint == '\n' {
		scanner.line += 1
	}

	return codePoint
}

func (scanner *Scanner) peek() rune {
	codePoint, width := utf8.DecodeRuneInString(scanner.source[scanner.current:])
	if width == 0 {
		return '\u0000'
	}

	return codePoint
}

func (scanner *Scanner) peekNext() rune {
	_, width := utf8.DecodeRuneInString(scanner.source[scanner.current:])
	if width == 0 {
		return '\u0000'
	}

	next := scanner.current + width
	codePoint, width := utf8.DecodeRuneInString(scanner.source[next:])
	if width == 0 {
		return '\u0000'
	}

	return codePoint
}

func (scanner *Scanner) isSensitive() bool {
	return scanner.sensitivity == 0
}

func (scanner *Scanner) sensitize() {
	if !scanner.isSensitive() {
		scanner.sensitivity--
	}
}

func (scanner *Scanner) desensitize() {
	scanner.sensitivity++
}

func (scanner *Scanner) createPos() logger.Pos {
	return logger.Pos{
		Start: scanner.start,
		End:   scanner.current,
		Line:  scanner.line,
	}
}

func (scanner *Scanner) addToken(kind TokenKind) {
	scanner.tokens = append(scanner.tokens, Token{
		Kind: kind,
		Pos:  scanner.createPos(),
	})
}

func (scanner *Scanner) addLiteralToken(kind TokenKind, lexeme string) {
	scanner.tokens = append(scanner.tokens, Token{
		Kind:  kind,
		Value: lexeme,
		Pos:   scanner.createPos(),
	})
}

func (scanner *Scanner) addError(message string) {
	scanner.logger.Add(message, scanner.createPos())
}

func isDigit(value rune) bool {
	return '0' <= value && value <= '9'
}

func isAlphabetic(value rune) bool {
	return ('A' <= value && value <= 'Z') ||
		('a' <= value && value <= 'z') ||
		value == '_'
}

func isAlphanumeric(value rune) bool {
	return isAlphabetic(value) || isDigit(value)
}
