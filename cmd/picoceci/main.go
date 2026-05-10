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
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kristofer/picoceci/pkg/ast"
	"github.com/kristofer/picoceci/pkg/bytecode"
	"github.com/kristofer/picoceci/pkg/eval"
	"github.com/kristofer/picoceci/pkg/lexer"
	"github.com/kristofer/picoceci/pkg/module"
	"github.com/kristofer/picoceci/pkg/object"
	"github.com/kristofer/picoceci/pkg/parser"
	"github.com/kristofer/picoceci/pkg/sdcard"
)

const version = "0.2.0-dev"

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
	case "run-vm":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: picoceci run-vm <file.pc>")
			os.Exit(1)
		}
		runFileVM(os.Args[2])
	case "repl":
		runREPL()
	case "repl-vm":
		runREPLVM()
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
	picoceci run-vm <file.pc> execute a picoceci source file via bytecode VM
  picoceci repl             start an interactive REPL
	picoceci repl-vm          start an interactive REPL via bytecode VM
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

// runFileVM reads, parses, compiles, and executes a .pc source file via bytecode VM.
func runFileVM(path string) {
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "picoceci: %v\n", err)
		os.Exit(1)
	}

	if _, err := execSourceVM(string(src), createModuleLoader(), nil, nil); err != nil {
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
//
// Paste mode: type "---" alone on a line to enter paste mode; all
// subsequent lines are buffered.  Type "---" again to execute the
// buffered program as a single unit.  This lets you paste multi-line
// programs without triggering a parse error on every incomplete line.
func runREPL() {
	fmt.Printf("picoceci %s  (type Ctrl-D to exit)\n", version)
	fmt.Println("  tip: type '---' to enter/exit paste mode for multi-line programs")
	runREPLWithIO(os.Stdin, os.Stdout, os.Stderr)
}

// runREPLVM starts a bytecode-VM-backed interactive REPL.
func runREPLVM() {
	fmt.Printf("picoceci %s (bytecode VM)  (type Ctrl-D to exit)\n", version)
	fmt.Println("  tip: type '---' to enter/exit paste mode for multi-line programs")
	runREPLWithVMIO(os.Stdin, os.Stdout, os.Stderr)
}

// runREPLWithIO is the testable core of the REPL.
// It reads from r, writes prompts/results to out, and errors to errOut.
func runREPLWithIO(r io.Reader, out, errOut io.Writer) {
	// Create interpreter with module loader for imports
	loader := createModuleLoader()
	interp := eval.NewWithLoader(loader)

	scanner := bufio.NewScanner(r)
	var buf strings.Builder
	inPaste := false

	for {
		if inPaste {
			fmt.Fprint(out, "... ")
		} else {
			fmt.Fprint(out, "picoceci> ")
		}
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()

		// "---" toggles paste mode.
		if line == "---" {
			if !inPaste {
				inPaste = true
				buf.Reset()
				fmt.Fprintln(out, "(paste mode on: type '---' to run)")
			} else {
				inPaste = false
				src := buf.String()
				buf.Reset()
				if src == "" {
					continue
				}
				evalSourceWithIO(interp, src, out, errOut)
			}
			continue
		}

		if inPaste {
			buf.WriteString(line)
			buf.WriteByte('\n')
			continue
		}

		if line == "" {
			continue
		}

		evalSourceWithIO(interp, line, out, errOut)
	}
	fmt.Fprintln(out)
}

// runREPLWithVMIO is the bytecode-VM counterpart of runREPLWithIO.
func runREPLWithVMIO(r io.Reader, out, errOut io.Writer) {
	loader := createModuleLoader()
	globals := make(map[string]*object.Object)
	blocks := make([]*bytecode.CompiledBlock, 0)

	scanner := bufio.NewScanner(r)
	var buf strings.Builder
	inPaste := false

	for {
		if inPaste {
			fmt.Fprint(out, "... ")
		} else {
			fmt.Fprint(out, "picoceci(vm)> ")
		}
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()

		// "---" toggles paste mode.
		if line == "---" {
			if !inPaste {
				inPaste = true
				buf.Reset()
				fmt.Fprintln(out, "(paste mode on: type '---' to run)")
			} else {
				inPaste = false
				src := buf.String()
				buf.Reset()
				if src == "" {
					continue
				}
				execSourceWithVMIO(loader, globals, &blocks, src, out, errOut)
			}
			continue
		}

		if inPaste {
			buf.WriteString(line)
			buf.WriteByte('\n')
			continue
		}

		if line == "" {
			continue
		}

		execSourceWithVMIO(loader, globals, &blocks, line, out, errOut)
	}
	fmt.Fprintln(out)
}

// evalSourceWithIO parses and evaluates src, writing results to out and
// errors to errOut.
func evalSourceWithIO(interp *eval.Interpreter, src string, out, errOut io.Writer) {
	prog, err := parseSource([]byte(src))
	if err != nil {
		fmt.Fprintf(errOut, "parse error: %v\n", err)
		return
	}

	result, err := interp.Eval(prog.Statements)
	if err != nil {
		fmt.Fprintf(errOut, "error: %v\n", err)
		return
	}
	if result != nil {
		fmt.Fprintln(out, "=>", result.PrintString())
	}
}

// execSourceWithVMIO parses, compiles, and executes src with the bytecode VM.
func execSourceWithVMIO(loader *module.Loader, globals map[string]*object.Object, blocks *[]*bytecode.CompiledBlock, src string, out, errOut io.Writer) {
	result, err := execSourceVM(src, loader, globals, blocks)
	if err != nil {
		fmt.Fprintf(errOut, "error: %v\n", err)
		return
	}
	if result != nil {
		fmt.Fprintln(out, "=>", result.PrintString())
	}
}

func execSourceVM(src string, loader *module.Loader, globals map[string]*object.Object, blocks *[]*bytecode.CompiledBlock) (*object.Object, error) {
	prog, err := parseSource([]byte(src))
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	c := bytecode.NewCompilerWithLoader(loader)
	if blocks != nil {
		c.SetBlocks(*blocks)
	}
	if globals != nil {
		c.SetTopLevelVarsAreGlobals(true)
	}
	chunk, err := c.Compile(prog.Statements)
	if err != nil {
		return nil, fmt.Errorf("compile error: %w", err)
	}

	vm := bytecode.NewVM()
	vm.SetBlocks(c.GetBlocks())
	if globals != nil {
		vm.AddGlobals(globals)
	}
	vm.AddGlobals(c.GetGlobals())

	result, err := vm.Run(chunk)
	if err != nil {
		return nil, err
	}

	if globals != nil {
		for name, val := range vm.Globals() {
			globals[name] = val
		}
	}
	if blocks != nil {
		*blocks = c.GetBlocks()
	}

	return result, nil
}

// parseSource tokenises and parses src, returning the AST or an error.
func parseSource(src []byte) (*ast.Program, error) {
	l := lexer.New(src)
	p := parser.New(l)
	return p.ParseProgram()
}
