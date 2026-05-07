package main

import (
	"bytes"
	"strings"
	"testing"
)

// TestREPLSingleLine verifies that a single-line expression is evaluated
// and its result printed in normal (non-paste) mode.
func TestREPLSingleLine(t *testing.T) {
	in := strings.NewReader("1 + 2.\n")
	var out, errOut bytes.Buffer

	runREPLWithIO(in, &out, &errOut)

	if errOut.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
	if !strings.Contains(out.String(), "=> 3") {
		t.Errorf("expected '=> 3' in output, got: %q", out.String())
	}
}

// TestREPLEmptyLines verifies that blank lines are ignored and do not cause
// errors in normal mode.
func TestREPLEmptyLines(t *testing.T) {
	in := strings.NewReader("\n\n1 + 1.\n\n")
	var out, errOut bytes.Buffer

	runREPLWithIO(in, &out, &errOut)

	if errOut.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
	if !strings.Contains(out.String(), "=> 2") {
		t.Errorf("expected '=> 2' in output, got: %q", out.String())
	}
}

// TestREPLPasteModeBasic verifies that "---" enters paste mode and the
// second "---" executes the buffered program as a single unit.
func TestREPLPasteModeBasic(t *testing.T) {
	// A two-line expression that only makes sense as a unit:
	// first line alone would fail to parse.
	input := "---\n| x: Int |\nx := 40 + 2.\nx.\n---\n"
	in := strings.NewReader(input)
	var out, errOut bytes.Buffer

	runREPLWithIO(in, &out, &errOut)

	if errOut.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
	if !strings.Contains(out.String(), "paste mode on") {
		t.Errorf("expected paste mode message in output, got: %q", out.String())
	}
	if !strings.Contains(out.String(), "=> 42") {
		t.Errorf("expected '=> 42' in output, got: %q", out.String())
	}
}

// TestREPLPasteModeEmptyBuffer verifies that "---" followed immediately by
// another "---" (empty paste block) executes nothing and does not error.
func TestREPLPasteModeEmptyBuffer(t *testing.T) {
	input := "---\n---\n1.\n"
	in := strings.NewReader(input)
	var out, errOut bytes.Buffer

	runREPLWithIO(in, &out, &errOut)

	if errOut.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
	if !strings.Contains(out.String(), "=> 1") {
		t.Errorf("expected '=> 1' in output after empty paste block, got: %q", out.String())
	}
}

// TestREPLPasteModeSyntaxError verifies that a syntax error in a pasted
// block is reported without crashing the REPL.
func TestREPLPasteModeSyntaxError(t *testing.T) {
	input := "---\nobject { bad syntax !!!\n---\n1.\n"
	in := strings.NewReader(input)
	var out, errOut bytes.Buffer

	runREPLWithIO(in, &out, &errOut)

	if errOut.Len() == 0 {
		t.Fatal("expected a parse error on stderr, got none")
	}
	// REPL should continue after the error and evaluate the next expression.
	if !strings.Contains(out.String(), "=> 1") {
		t.Errorf("expected '=> 1' after parse error recovery, got output: %q", out.String())
	}
}

// TestREPLPasteModeMultiline verifies that a real multi-line program (an
// object declaration with methods) evaluates correctly when pasted.
func TestREPLPasteModeMultiline(t *testing.T) {
	program := `object Counter {
    | count: Int |
    init  [ count := 0 ]
    inc   [ count := count + 1. ^self ]
    value [ ^count ]
}
| c: Counter |
c := Counter new.
c inc; inc; inc.
c value.
`
	input := "---\n" + program + "---\n"
	in := strings.NewReader(input)
	var out, errOut bytes.Buffer

	runREPLWithIO(in, &out, &errOut)

	if errOut.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
	if !strings.Contains(out.String(), "=> 3") {
		t.Errorf("expected '=> 3' from counter.value, got: %q", out.String())
	}
}

// TestREPLVMSingleLine verifies that the VM-backed REPL can evaluate
// a simple expression on the host.
func TestREPLVMSingleLine(t *testing.T) {
	in := strings.NewReader("1 + 2.\n")
	var out, errOut bytes.Buffer

	runREPLWithVMIO(in, &out, &errOut)

	if errOut.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
	if !strings.Contains(out.String(), "=> 3") {
		t.Errorf("expected '=> 3' in output, got: %q", out.String())
	}
}
