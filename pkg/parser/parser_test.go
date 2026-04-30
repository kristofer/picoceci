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
