package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/HereIsKevin/edible/internal/logger"
	"github.com/HereIsKevin/edible/internal/parser"
	"github.com/HereIsKevin/edible/internal/scanner"
)

func main() {
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile := flag.String("memprofile", "", "write memory profile to file")
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
		fmt.Println(tokens)
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

	if *memprofile != "" {
		file, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()

		runtime.GC()
		if err := pprof.WriteHeapProfile(file); err != nil {
			log.Fatal(err)
		}
	}
}
