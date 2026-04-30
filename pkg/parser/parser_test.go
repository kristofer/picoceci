package parser_test

import (
	"testing"

	"github.com/kristofer/picoceci/pkg/ast"
	"github.com/kristofer/picoceci/pkg/lexer"
	"github.com/kristofer/picoceci/pkg/parser"
)

func parse(t *testing.T, src string) *ast.Program {
	t.Helper()
	l := lexer.NewString(src)
	p := parser.New(l)
	prog, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return prog
}

func TestParser_IntegerLiteral(t *testing.T) {
	prog := parse(t, "42.")
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}
	lit, ok := prog.Statements[0].(*ast.IntLit)
	if !ok {
		t.Fatalf("expected *ast.IntLit, got %T", prog.Statements[0])
	}
	if lit.Value != 42 {
		t.Errorf("value: got %d, want 42", lit.Value)
	}
}

func TestParser_FloatLiteral(t *testing.T) {
	prog := parse(t, "3.14.")
	lit, ok := prog.Statements[0].(*ast.FloatLit)
	if !ok {
		t.Fatalf("expected *ast.FloatLit, got %T", prog.Statements[0])
	}
	if lit.Value != 3.14 {
		t.Errorf("value: got %f, want 3.14", lit.Value)
	}
}

func TestParser_StringLiteral(t *testing.T) {
	prog := parse(t, "'hello'.")
	lit, ok := prog.Statements[0].(*ast.StringLit)
	if !ok {
		t.Fatalf("expected *ast.StringLit, got %T", prog.Statements[0])
	}
	if lit.Value != "hello" {
		t.Errorf("value: got %q, want \"hello\"", lit.Value)
	}
}

func TestParser_BoolLiterals(t *testing.T) {
	for _, src := range []string{"true.", "false."} {
		prog := parse(t, src)
		if _, ok := prog.Statements[0].(*ast.BoolLit); !ok {
			t.Errorf("%s: expected *ast.BoolLit, got %T", src, prog.Statements[0])
		}
	}
}

func TestParser_NilLiteral(t *testing.T) {
	prog := parse(t, "nil.")
	if _, ok := prog.Statements[0].(*ast.NilLit); !ok {
		t.Fatalf("expected *ast.NilLit, got %T", prog.Statements[0])
	}
}

func TestParser_UnaryMessage(t *testing.T) {
	prog := parse(t, "42 factorial.")
	msg, ok := prog.Statements[0].(*ast.UnaryMsg)
	if !ok {
		t.Fatalf("expected *ast.UnaryMsg, got %T", prog.Statements[0])
	}
	if msg.Selector != "factorial" {
		t.Errorf("selector: got %q, want \"factorial\"", msg.Selector)
	}
}

func TestParser_BinaryMessage(t *testing.T) {
	prog := parse(t, "3 + 4.")
	msg, ok := prog.Statements[0].(*ast.BinaryMsg)
	if !ok {
		t.Fatalf("expected *ast.BinaryMsg, got %T", prog.Statements[0])
	}
	if msg.Op != "+" {
		t.Errorf("op: got %q, want \"+\"", msg.Op)
	}
}

func TestParser_KeywordMessage(t *testing.T) {
	prog := parse(t, "dict at: #key put: 42.")
	msg, ok := prog.Statements[0].(*ast.KeywordMsg)
	if !ok {
		t.Fatalf("expected *ast.KeywordMsg, got %T", prog.Statements[0])
	}
	if msg.Selector() != "at:put:" {
		t.Errorf("selector: got %q, want \"at:put:\"", msg.Selector())
	}
	if len(msg.Args) != 2 {
		t.Errorf("args: got %d, want 2", len(msg.Args))
	}
}

func TestParser_Assignment(t *testing.T) {
	prog := parse(t, "x := 42.")
	assign, ok := prog.Statements[0].(*ast.Assign)
	if !ok {
		t.Fatalf("expected *ast.Assign, got %T", prog.Statements[0])
	}
	if assign.Name != "x" {
		t.Errorf("name: got %q, want \"x\"", assign.Name)
	}
}

func TestParser_VarDecl(t *testing.T) {
	prog := parse(t, "| x y z |")
	vd, ok := prog.Statements[0].(*ast.VarDecl)
	if !ok {
		t.Fatalf("expected *ast.VarDecl, got %T", prog.Statements[0])
	}
	if len(vd.Names) != 3 {
		t.Errorf("names: got %v, want [x y z]", vd.Names)
	}
}

func TestParser_Block(t *testing.T) {
	prog := parse(t, "[ :x | x + 1 ].")
	blk, ok := prog.Statements[0].(*ast.Block)
	if !ok {
		t.Fatalf("expected *ast.Block, got %T", prog.Statements[0])
	}
	if len(blk.Params) != 1 || blk.Params[0] != "x" {
		t.Errorf("params: got %v, want [x]", blk.Params)
	}
	if len(blk.Body) != 1 {
		t.Errorf("body: got %d statements, want 1", len(blk.Body))
	}
}

func TestParser_Return(t *testing.T) {
	prog := parse(t, "^42.")
	ret, ok := prog.Statements[0].(*ast.Return)
	if !ok {
		t.Fatalf("expected *ast.Return, got %T", prog.Statements[0])
	}
	if _, ok := ret.Value.(*ast.IntLit); !ok {
		t.Errorf("return value: expected *ast.IntLit, got %T", ret.Value)
	}
}

func TestParser_Cascade(t *testing.T) {
	prog := parse(t, "stream nextPut: $a; nextPut: $b.")
	_, ok := prog.Statements[0].(*ast.Cascade)
	if !ok {
		t.Fatalf("expected *ast.Cascade, got %T", prog.Statements[0])
	}
}

func TestParser_ObjectDecl(t *testing.T) {
	src := `
object Counter {
    | count |
    init [ count := 0 ]
    inc  [ count := count + 1. ^self ]
    value [ ^count ]
}`
	prog := parse(t, src)
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}
	decl, ok := prog.Statements[0].(*ast.ObjectDecl)
	if !ok {
		t.Fatalf("expected *ast.ObjectDecl, got %T", prog.Statements[0])
	}
	if decl.Name != "Counter" {
		t.Errorf("name: got %q, want \"Counter\"", decl.Name)
	}
	if len(decl.Slots) != 1 || decl.Slots[0] != "count" {
		t.Errorf("slots: got %v, want [count]", decl.Slots)
	}
	if len(decl.Methods) != 3 {
		t.Errorf("methods: got %d, want 3", len(decl.Methods))
	}
}

func TestParser_InterfaceDecl(t *testing.T) {
	src := `
interface Incrementable {
    inc
    dec
    value
}`
	prog := parse(t, src)
	decl, ok := prog.Statements[0].(*ast.InterfaceDecl)
	if !ok {
		t.Fatalf("expected *ast.InterfaceDecl, got %T", prog.Statements[0])
	}
	if decl.Name != "Incrementable" {
		t.Errorf("name: got %q, want \"Incrementable\"", decl.Name)
	}
	if len(decl.Sigs) != 3 {
		t.Errorf("sigs: got %d, want 3", len(decl.Sigs))
	}
}

func TestParser_ImportDecl(t *testing.T) {
	prog := parse(t, "import 'Counter'.")
	imp, ok := prog.Statements[0].(*ast.ImportDecl)
	if !ok {
		t.Fatalf("expected *ast.ImportDecl, got %T", prog.Statements[0])
	}
	if imp.Path != "Counter" {
		t.Errorf("path: got %q, want \"Counter\"", imp.Path)
	}
}

func TestParser_SymbolLiteral(t *testing.T) {
	prog := parse(t, "#hello.")
	sym, ok := prog.Statements[0].(*ast.SymbolLit)
	if !ok {
		t.Fatalf("expected *ast.SymbolLit, got %T", prog.Statements[0])
	}
	if sym.Value != "hello" {
		t.Errorf("value: got %q, want \"hello\"", sym.Value)
	}
}

func TestParser_HexInteger(t *testing.T) {
	prog := parse(t, "16rFF.")
	lit, ok := prog.Statements[0].(*ast.IntLit)
	if !ok {
		t.Fatalf("expected *ast.IntLit, got %T", prog.Statements[0])
	}
	if lit.Value != 255 {
		t.Errorf("value: got %d, want 255", lit.Value)
	}
}

func TestParser_NoPanicOnEmptyInput(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("parser panicked on empty input: %v", r)
		}
	}()
	parse(t, "")
}

func TestParser_NoPanicOnGarbage(t *testing.T) {
	inputs := []string{")", "]", "}", ":::", "^^^"}
	for _, in := range inputs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("parser panicked on %q: %v", in, r)
				}
			}()
			l := lexer.NewString(in)
			p := parser.New(l)
			p.ParseProgram() //nolint:errcheck
		}()
	}
}

// TestParser_ErrorRecovery_UnbalancedBracket verifies a meaningful error for `[`.
func TestParser_ErrorRecovery_UnbalancedBracket(t *testing.T) {
	l := lexer.NewString("[ 42")
	p := parser.New(l)
	_, err := p.ParseProgram()
	if err == nil {
		t.Error("expected parse error for unbalanced [, got nil")
	}
}

// TestParser_ErrorRecovery_MissingDot verifies that a missing statement
// terminator does not cause a panic.
func TestParser_ErrorRecovery_MissingDot(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("parser panicked on missing dot: %v", r)
		}
	}()
	l := lexer.NewString("42")
	p := parser.New(l)
	prog, _ := p.ParseProgram()
	if len(prog.Statements) != 1 {
		t.Errorf("expected 1 statement without trailing dot, got %d", len(prog.Statements))
	}
}

// TestParser_LanguageSpec_Examples round-trips all significant code examples
// from LANGUAGE_SPEC.md to confirm they parse without error.
func TestParser_LanguageSpec_Examples(t *testing.T) {
	examples := []struct {
		name string
		src  string
	}{
		// §2.5 Literals
		{"integer decimal", "42."},
		{"integer hex", "16rFF."},
		{"integer binary", "2r1010."},
		{"float", "3.14."},
		{"float exp", "1.5e-3."},
		{"character", "$A."},
		{"string", "'Hello'."},
		{"string escaped quote", "'it''s fine'."},
		{"symbol ident", "#hello."},
		{"symbol keyword", "#at:put:."},
		{"symbol quoted", "#'with spaces'."},
		{"byte array", "#[1 2 3 255]."},
		{"array literal", "#(1 'two' #three)."},
		{"bool true", "true."},
		{"bool false", "false."},
		{"nil", "nil."},
		// §4 Expressions
		{"unary message", "42 factorial."},
		{"unary chain", "'hello' reversed."},
		{"binary message", "3 + 4."},
		{"keyword message one arg", "collection at: 2."},
		{"keyword message two args", "dict at: #key put: value."},
		{"assignment", "| x y |\nx := 42.\ny := x + 1."},
		{"cascade", "Transcript\n    print: 'a';\n    print: 'b';\n    nl."},
		{"return", "^value."},
		{"self", "self."},
		{"super", "super."},
		{"thisContext", "thisContext."},
		{"parenthesised", "(3 + 4) * 2."},
		// §5.1 Object declaration
		{"object Counter", `
object Counter {
    | count |
    init [ count := 0 ]
    inc  [ count := count + 1. ^self ]
    dec  [ count := count - 1. ^self ]
    value [ ^count ]
    printString [ ^'Counter(', count printString, ')' ]
}`},
		{"object creating instances", "| c |\nc := Counter new."},
		{"object composition", `
object LoggedCounter {
    compose Counter.
    inc [
        count := count + 1.
        Console println: 'incremented to ', count printString.
        ^self
    ]
}`},
		{"anonymous object literal", "| point |\npoint := object { x := 3. y := 4 }."},
		// §6 Interfaces
		{"interface", `
interface Incrementable {
    inc
    dec
    value
}`},
		// §7 Control flow
		{"ifTrue:ifFalse:", "x > 0\n    ifTrue:  [ Console println: 'positive' ]\n    ifFalse: [ Console println: 'non-positive' ]."},
		{"ifTrue: with return", "(x = 0)\n    ifTrue: [ ^0 ]."},
		{"to:do:", "1 to: 10 do: [ :i | Console println: i printString ]."},
		{"whileTrue:", "[ x > 0 ] whileTrue: [ x := x - 1 ]."},
		{"whileFalse:", "[ x < 0 ] whileFalse: [ x := x + 1 ]."},
		{"timesRepeat:", "5 timesRepeat: [ Console println: 'tick' ]."},
		{"do:", "#(1 2 3) do: [ :each | Console println: each printString ]."},
		{"collect:", "| doubled |\ndoubled := #(1 2 3) collect: [ :each | each * 2 ]."},
		{"inject:into:", "| sum |\nsum := #(1 2 3) inject: 0 into: [ :acc :each | acc + each ]."},
		// §8 Blocks
		{"block two params", "[ :x :y | x + y ]."},
		{"block closure", "| adder |\nadder := [ :n | [ :x | x + n ] ].\n(adder value: 5) value: 3."},
		// §9 Error handling
		{"on:do:", "[ someRiskyOperation ]\n    on: Error\n    do: [ :err | Console println: err messageText ]."},
		{"ensure:", "[ file read ]\n    ensure: [ file close ]."},
		// import
		{"import", "import 'Counter'."},
	}

	for _, ex := range examples {
		ex := ex
		t.Run(ex.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("parser panicked: %v", r)
				}
			}()
			l := lexer.NewString(ex.src)
			p := parser.New(l)
			prog, err := p.ParseProgram()
			if err != nil {
				t.Errorf("parse error: %v", err)
			}
			if prog == nil {
				t.Error("got nil program")
			}
		})
	}
}

// TestParser_AnonObjectLit verifies anonymous object literal parsing.
func TestParser_AnonObjectLit(t *testing.T) {
	prog := parse(t, "object { x := 3. y := 4 }.")
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}
	anon, ok := prog.Statements[0].(*ast.AnonObjectLit)
	if !ok {
		t.Fatalf("expected *ast.AnonObjectLit, got %T", prog.Statements[0])
	}
	if len(anon.Slots) != 2 {
		t.Errorf("expected 2 slots, got %d", len(anon.Slots))
	}
	if anon.Slots[0].Name != "x" {
		t.Errorf("slot[0].Name: got %q, want \"x\"", anon.Slots[0].Name)
	}
	if anon.Slots[1].Name != "y" {
		t.Errorf("slot[1].Name: got %q, want \"y\"", anon.Slots[1].Name)
	}
}

// TestParser_ThisContext verifies that thisContext parses as a primary.
func TestParser_ThisContext(t *testing.T) {
	prog := parse(t, "thisContext.")
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}
	if _, ok := prog.Statements[0].(*ast.ThisContextExpr); !ok {
		t.Fatalf("expected *ast.ThisContextExpr, got %T", prog.Statements[0])
	}
}
