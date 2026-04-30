// Command picoceci is the desktop host for the picoceci interpreter.
//
// Usage:
//
//	picoceci run <file.pc>    — execute a picoceci source file
//	picoceci repl             — start an interactive REPL
//	picoceci version          — print version information
package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/kristofer/picoceci/pkg/ast"
	"github.com/kristofer/picoceci/pkg/eval"
	"github.com/kristofer/picoceci/pkg/lexer"
	"github.com/kristofer/picoceci/pkg/parser"
)

const version = "0.1.0-dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "run":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: picoceci run <file.pc>")
			os.Exit(1)
		}
		runFile(os.Args[2])
	case "repl":
		runREPL()
	case "version":
		fmt.Printf("picoceci %s\n", version)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `picoceci — a small, high-protein language (v%s)

Usage:
  picoceci run <file.pc>    execute a picoceci source file
  picoceci repl             start an interactive REPL
  picoceci version          print version information
`, version)
}

// runFile reads, parses, and evaluates a .pc source file.
func runFile(path string) {
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "picoceci: %v\n", err)
		os.Exit(1)
	}

	prog, err := parseSource(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "picoceci: %v\n", err)
		os.Exit(1)
	}

	interp := eval.New()
	_, err = interp.Eval(prog.Statements)
	if err != nil {
		fmt.Fprintf(os.Stderr, "picoceci: runtime error: %v\n", err)
		os.Exit(1)
	}
}

// runREPL starts an interactive Read-Eval-Print Loop.
func runREPL() {
	fmt.Printf("picoceci %s  (type Ctrl-D to exit)\n", version)
	interp := eval.New()
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("picoceci> ")
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		if line == "" {
			continue
		}

		prog, err := parseSource([]byte(line))
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
			continue
		}

		result, err := interp.Eval(prog.Statements)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}
		if result != nil {
			fmt.Println("=>", result.PrintString())
		}
	}
	fmt.Println()
}

// parseSource tokenises and parses src, returning the AST or an error.
func parseSource(src []byte) (*ast.Program, error) {
	l := lexer.New(src)
	p := parser.New(l)
	return p.ParseProgram()
}
