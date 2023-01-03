package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/HereIsKevin/edible/internal/logger"
	"github.com/HereIsKevin/edible/internal/parser"
	"github.com/HereIsKevin/edible/internal/scanner"
)

func main() {
	contents, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	source := string(contents)
	logger := logger.New(source)

	scannerStart := time.Now()
	tokens := scanner.New(source, logger).Scan()
	scannerEnd := float64(time.Since(scannerStart)) / float64(time.Millisecond)

	fmt.Println("========== SCANNER:", scannerEnd, "ms ==========")

	if logger.Log() {
		os.Exit(1)
	}

	indent := 0

	for _, token := range tokens {
		switch token.Kind {
		case scanner.TokenOpenParen,
			scanner.TokenOpenBrack,
			scanner.TokenOpenBrace,
			scanner.TokenOpenBlock:
			indent += 1
			fmt.Print(token, "\n", strings.Repeat("    ", indent))
		case scanner.TokenCloseParen,
			scanner.TokenCloseBrack,
			scanner.TokenCloseBrace,
			scanner.TokenCloseBlock:
			indent -= 1
			fmt.Print("\n", strings.Repeat("    ", indent), token, " ")
		case scanner.TokenEOF, scanner.TokenComma, scanner.TokenNewline:
			fmt.Print(token, "\n", strings.Repeat("    ", indent))
		default:
			fmt.Print(token, " ")
		}
	}

	parserStart := time.Now()
	expr := parser.New(tokens, logger).Parse()
	parserEnd := float64(time.Since(parserStart)) / float64(time.Millisecond)

	fmt.Println("\n========== PARSER:", parserEnd, "ms ==========")

	if logger.Log() {
		os.Exit(1)
	}

	fmt.Println(expr)
}
