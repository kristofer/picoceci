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
	"path/filepath"

	"github.com/kristofer/picoceci/pkg/ast"
	"github.com/kristofer/picoceci/pkg/eval"
	"github.com/kristofer/picoceci/pkg/lexer"
	"github.com/kristofer/picoceci/pkg/module"
	"github.com/kristofer/picoceci/pkg/parser"
	"github.com/kristofer/picoceci/pkg/sdcard"
)

const version = "0.1.0-dev"

func main() {
	// Initialize SD card stub for desktop (maps /sdcard/ to ./testdata/sdcard/)
	initSDCard()

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

// initSDCard sets up the SD card stub for desktop use.
// This allows picoceci scripts to access /sdcard/ paths.
func initSDCard() {
	// Find testdata/sdcard relative to executable or working directory
	candidates := []string{
		"./testdata/sdcard",
		filepath.Join(filepath.Dir(os.Args[0]), "testdata", "sdcard"),
	}

	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			sdcard.SetRoot(path)
			break
		}
	}

	// Mount silently - if it fails, imports from /sdcard/ just won't work
	_ = sdcard.Mount("/sdcard/")
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

	// Create module loader with SD card file reader
	loader := createModuleLoader()
	interp := eval.NewWithLoader(loader)

	_, err = interp.Eval(prog.Statements)
	if err != nil {
		fmt.Fprintf(os.Stderr, "picoceci: runtime error: %v\n", err)
		os.Exit(1)
	}
}

// createModuleLoader creates a module loader that can read from
// both the local filesystem and the SD card stub.
func createModuleLoader() *module.Loader {
	// Create a file reader that tries SD card first, then local filesystem
	reader := func(path string) ([]byte, error) {
		// Try SD card path first
		if sdcard.IsMounted() {
			if data, err := sdcard.ReadFile(path); err == nil {
				return data, nil
			}
		}
		// Fall back to local filesystem
		return os.ReadFile(path)
	}

	resolver := module.NewResolver(reader)
	module.RegisterBuiltins(resolver)
	return module.NewLoader(resolver)
}

// runREPL starts an interactive Read-Eval-Print Loop.
func runREPL() {
	fmt.Printf("picoceci %s  (type Ctrl-D to exit)\n", version)

	// Create interpreter with module loader for imports
	loader := createModuleLoader()
	interp := eval.NewWithLoader(loader)

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
