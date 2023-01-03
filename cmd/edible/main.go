package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/HereIsKevin/edible/internal/logger"
	"github.com/HereIsKevin/edible/internal/parser"
	"github.com/HereIsKevin/edible/internal/scanner"
)

func main() {
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	silent := flag.Bool("silent", false, "disable debug output")
	flag.Parse()

	if *cpuprofile != "" {
		file, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		if err := pprof.StartCPUProfile(file); err != nil {
			log.Fatal(err)
		}
		defer pprof.StopCPUProfile()
	}

	path := flag.Arg(0)
	contents, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	source := string(contents)
	logger := logger.New(source)

	scannerStart := time.Now()
	tokens := scanner.New(source, logger).Scan()
	scannerEnd := float64(time.Since(scannerStart)) / float64(time.Millisecond)

	fmt.Printf("========== SCANNER: %f ms (%d tokens) ==========\n", scannerEnd, len(tokens))

	if logger.Log() {
		os.Exit(1)
	}

	if !*silent {
		indent := 0
		builder := strings.Builder{}

		for _, token := range tokens {
			var value string

			switch token.Kind {
			case scanner.TokenOpenParen,
				scanner.TokenOpenBrack,
				scanner.TokenOpenBrace,
				scanner.TokenOpenBlock:
				indent += 1
				value = fmt.Sprintf("%s\n%s", token, strings.Repeat("    ", indent))
			case scanner.TokenCloseParen,
				scanner.TokenCloseBrack,
				scanner.TokenCloseBrace,
				scanner.TokenCloseBlock:
				indent -= 1
				value = fmt.Sprintf("\n%s%s ", strings.Repeat("    ", indent), token)
			case scanner.TokenEOF, scanner.TokenComma, scanner.TokenNewline:
				value = fmt.Sprintf("%s\n%s", token, strings.Repeat("    ", indent))
			default:
				value = fmt.Sprintf("%s ", token)
			}

			builder.WriteString(value)
		}

		fmt.Println(builder.String())
	}

	parserStart := time.Now()
	expr := parser.New(tokens, logger).Parse()
	parserEnd := float64(time.Since(parserStart)) / float64(time.Millisecond)

	fmt.Printf("========== PARSER: %f ms ==========\n", parserEnd)

	if logger.Log() {
		os.Exit(1)
	}

	if !*silent {
		fmt.Println(expr)
	}
}
