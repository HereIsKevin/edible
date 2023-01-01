package scanner

import (
	"unicode"
	"unicode/utf8"

	"github.com/HereIsKevin/edible/internal/logger"
)

type TokenKind int

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

type Token struct {
	Kind   TokenKind
	Lexeme string
	Span   logger.Span
}

// TODO: Ternary operator.
// TODO: Bitwise operators.
// TODO: Boolean operators.
// TODO: Floor division operator.
// TODO: Basic builtin functions.
type Scanner struct {
	source string

	tokens []Token
	errors []logger.Error

	indents     []int
	sensitivity int
	isLineStart bool

	previous rune
	start    int
	current  int
}

func New(source string) *Scanner {
	return &Scanner{
		source: source,

		tokens: []Token{},
		errors: []logger.Error{},

		indents:     []int{},
		sensitivity: 0,
		isLineStart: false,

		previous: '\u0000',
		start:    0,
		current:  0,
	}
}

func (scanner *Scanner) Scan() ([]Token, []logger.Error) {
	for !scanner.isEOF() {
		scanner.start = scanner.current
		scanner.scanToken()
	}

	// Auto-close all blocks by adding a dedent for every indent.
	for index := 0; index < len(scanner.indents); index++ {
		scanner.createToken(TokenCloseBlock)
	}

	// Add final EOF token
	scanner.createToken(TokenEOF)

	return scanner.tokens, scanner.errors
}

func (scanner *Scanner) scanToken() {
	character := scanner.advance()
	isLineStart := scanner.isLineStart
	scanner.isLineStart = false

	switch character {
	// Controls
	case ':':
		scanner.createToken(TokenColon)
	case ',':
		scanner.createToken(TokenComma)
	case '-':
		if isLineStart && scanner.isSensitive() {
			// Dashes at the start of a line in whitespace-sensitive areas denote arrays.
			scanner.scanDash()
		} else {
			// Otherwise, they are just a normal minus operator.
			scanner.createToken(TokenMinus)
		}

	// Operators
	case '+':
		scanner.createToken(TokenPlus)
	// See controls for minus operator.
	case '*':
		scanner.createToken(TokenStar)
	case '/':
		scanner.createToken(TokenSlash)
	case '<':
		scanner.createToken(TokenLess)
	case '.':
		scanner.createToken(TokenDot)
	case '$':
		scanner.createToken(TokenDollar)

	// Delimiters
	case '(':
		scanner.desensitize()
		scanner.createToken(TokenOpenParen)
	case ')':
		scanner.sensitize()
		scanner.createToken(TokenCloseParen)
	case '[':
		scanner.desensitize()
		scanner.createToken(TokenOpenBrack)
	case ']':
		scanner.sensitize()
		scanner.createToken(TokenCloseBrack)
	case '{':
		scanner.desensitize()
		scanner.createToken(TokenOpenBrace)
	case '}':
		scanner.sensitize()
		scanner.createToken(TokenCloseBrace)

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
			scanner.scanBlock()
		}

	case '\t':
		if scanner.isSensitive() {
			// Tabs are never valid as indentation, and are only ignored while not sensitive
			// to whitespace. This is because though they count as one level while scanning,
			// the same as spaces, text editors usually display them as 4 or 8 spaces.
			scanner.createError("Unexpected tab, only spaces are valid indentation.")
		}

	case '\r', '\n':
		if scanner.isSensitive() {
			// Handle LF and CRLF line endings while sensitive to whitespace.
			scanner.scanBlock()
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
				scanner.createError("Unexpected whitespace.")
			}
		} else {
			// Everything else, including random symbols, are invalid.
			scanner.createError("Unexpected character.")
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
	scanner.createToken(TokenDash)
	scanner.createToken(TokenOpenBlock)

	scanner.indents = append(scanner.indents, indent)
}

func (scanner *Scanner) scanComment() {
	for scanner.peek() != '\r' && scanner.peek() != '\n' && !scanner.isEOF() {
		// Ignore everything until a newline or the end.
		scanner.advance()
	}
}

// TODO: Maybe support tabs as indentation?
func (scanner *Scanner) scanBlock() {
	indent := 0
	newline := false

	if scanner.peekPrevious() == ' ' {
		indent += 1
	}

	if scanner.peekPrevious() == '\n' {
		newline = true
	}

loop:
	for {
		switch scanner.peek() {
		case ' ':
			scanner.advance()
			indent += 1
		case '\r':
			// Consume CR
			scanner.advance()

			if scanner.peek() != '\n' {
				// CR-only line endings are invalid while sensitive to whitespace. However,
				// instead of going crazy, just skip over it to recover.
				scanner.createError("Unexpected CR, line endings are CRLF and LF.")
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
		scanner.createToken(TokenOpenBlock)
	} else if indent < lastIndent {
		for len(scanner.indents) > 0 &&
			indent < scanner.indents[len(scanner.indents)-1] {
			// Close blocks until indent is at the correct level.
			scanner.indents = scanner.indents[:len(scanner.indents)-1]
			scanner.createToken(TokenCloseBlock)
		}

		// Dedenting always involves a newline
		scanner.createToken(TokenNewline)
	} else if newline && scanner.start == 0 {
		// Add a newline if needed unless it is a leading newline
		scanner.createToken(TokenNewline)
	}

	// Mark the current state as the start of the line since indentation only occurs
	// at the start of lines. This allows leading dashes to be handled properly as
	// block arrays.
	scanner.isLineStart = true
}

// TODO: Multiline strings.
// TODO: Literal strings.
// TODO: Escapes codes.
func (scanner *Scanner) scanString() {
	for scanner.peek() != '"' && !scanner.isEOF() {
		if scanner.peek() == '\n' {
			scanner.createError("Unexpected newline within string.")
			return
		}

		scanner.advance()
	}

	if scanner.isEOF() {
		scanner.createError("Unterminated string.")
		return
	}

	// Consume closing '"'.
	scanner.advance()

	// This is always valid since ASCII characters should only occupy one byte.
	value := scanner.source[scanner.start+1 : scanner.current-1]

	// Create string token.
	scanner.createLiteralToken(TokenStr, value)
}

func (scanner *Scanner) scanIdentifier() {
	for isAlphanumeric(scanner.peek()) {
		scanner.advance()
	}

	// Take value from source.
	value := scanner.source[scanner.start:scanner.current]

	// Create identifier token.
	scanner.createLiteralToken(TokenIdent, value)
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
		scanner.createLiteralToken(TokenFloat, value)
	} else {
		// Take value from source.
		value := scanner.source[scanner.start:scanner.current]

		// Create integer token.
		scanner.createLiteralToken(TokenInt, value)
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

	scanner.previous = codePoint
	scanner.current += width

	return codePoint
}

func (scanner *Scanner) peekPrevious() rune {
	return scanner.previous
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

func (scanner *Scanner) createSpan() logger.Span {
	return logger.Span{
		Start: scanner.start,
		End:   scanner.current,
	}
}

func (scanner *Scanner) createToken(kind TokenKind) {
	scanner.tokens = append(scanner.tokens, Token{
		Kind: kind,
		Span: scanner.createSpan(),
	})
}

func (scanner *Scanner) createLiteralToken(kind TokenKind, lexeme string) {
	scanner.tokens = append(scanner.tokens, Token{
		Kind:   kind,
		Lexeme: lexeme,
		Span:   scanner.createSpan(),
	})
}

func (scanner *Scanner) createError(message string) {
	scanner.errors = append(scanner.errors, logger.Error{
		Message: message,
		Span:    scanner.createSpan(),
	})
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
