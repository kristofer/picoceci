package bytecode

import (
	"strings"
	"testing"

	"github.com/kristofer/picoceci/pkg/lexer"
	"github.com/kristofer/picoceci/pkg/parser"
)

func compileSource(src string) (*Chunk, error) {
	l := lexer.NewString(src)
	p := parser.New(l)
	prog, err := p.ParseProgram()
	if err != nil {
		return nil, err
	}

	c := NewCompiler()
	return c.Compile(prog.Statements)
}

func TestCompileIntLiteral(t *testing.T) {
	chunk, err := compileSource("42.")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	dis := chunk.Disassemble("test")
	if !strings.Contains(dis, "PUSH_INT") {
		t.Errorf("expected PUSH_INT in disassembly:\n%s", dis)
	}
}

func TestCompileFloatLiteral(t *testing.T) {
	chunk, err := compileSource("3.14.")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	dis := chunk.Disassemble("test")
	if !strings.Contains(dis, "PUSH_CONST") {
		t.Errorf("expected PUSH_CONST in disassembly:\n%s", dis)
	}
}

func TestCompileStringLiteral(t *testing.T) {
	chunk, err := compileSource("'hello'.")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	dis := chunk.Disassemble("test")
	if !strings.Contains(dis, "PUSH_CONST") || !strings.Contains(dis, "'hello'") {
		t.Errorf("expected PUSH_CONST with 'hello' in disassembly:\n%s", dis)
	}
}

func TestCompileSymbolLiteral(t *testing.T) {
	chunk, err := compileSource("#hello.")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	dis := chunk.Disassemble("test")
	if !strings.Contains(dis, "PUSH_CONST") || !strings.Contains(dis, "#hello") {
		t.Errorf("expected PUSH_CONST with #hello in disassembly:\n%s", dis)
	}
}

func TestCompileBoolLiterals(t *testing.T) {
	chunk, err := compileSource("true.")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	dis := chunk.Disassemble("test")
	if !strings.Contains(dis, "PUSH_TRUE") {
		t.Errorf("expected PUSH_TRUE in disassembly:\n%s", dis)
	}

	chunk, err = compileSource("false.")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	dis = chunk.Disassemble("test")
	if !strings.Contains(dis, "PUSH_FALSE") {
		t.Errorf("expected PUSH_FALSE in disassembly:\n%s", dis)
	}
}

func TestCompileNilLiteral(t *testing.T) {
	chunk, err := compileSource("nil.")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	dis := chunk.Disassemble("test")
	if !strings.Contains(dis, "PUSH_NIL") {
		t.Errorf("expected PUSH_NIL in disassembly:\n%s", dis)
	}
}

func TestCompileBinaryMessage(t *testing.T) {
	chunk, err := compileSource("3 + 4.")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	dis := chunk.Disassemble("test")
	if !strings.Contains(dis, "PUSH_INT") {
		t.Errorf("expected PUSH_INT in disassembly:\n%s", dis)
	}
	if !strings.Contains(dis, "SEND") {
		t.Errorf("expected SEND in disassembly:\n%s", dis)
	}
}

func TestCompileUnaryMessage(t *testing.T) {
	chunk, err := compileSource("42 negated.")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	dis := chunk.Disassemble("test")
	if !strings.Contains(dis, "PUSH_INT") {
		t.Errorf("expected PUSH_INT in disassembly:\n%s", dis)
	}
	if !strings.Contains(dis, "SEND") || !strings.Contains(dis, "'negated'") {
		t.Errorf("expected SEND with 'negated' in disassembly:\n%s", dis)
	}
}

func TestCompileKeywordMessage(t *testing.T) {
	chunk, err := compileSource("1 to: 10 do: [ :i | i ].")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	dis := chunk.Disassemble("test")
	if !strings.Contains(dis, "SEND") || !strings.Contains(dis, "'to:do:'") {
		t.Errorf("expected SEND with 'to:do:' in disassembly:\n%s", dis)
	}
}

func TestCompileVarDecl(t *testing.T) {
	chunk, err := compileSource("| x: Any | x := 42.")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	dis := chunk.Disassemble("test")
	if !strings.Contains(dis, "STORE_LOCAL") {
		t.Errorf("expected STORE_LOCAL in disassembly:\n%s", dis)
	}
}

func TestCompileLocalVariable(t *testing.T) {
	chunk, err := compileSource("| x: Any | x := 42. x.")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	dis := chunk.Disassemble("test")
	if !strings.Contains(dis, "PUSH_LOCAL") {
		t.Errorf("expected PUSH_LOCAL in disassembly:\n%s", dis)
	}
}

func TestCompileGlobalVariable(t *testing.T) {
	chunk, err := compileSource("Console.")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	dis := chunk.Disassemble("test")
	if !strings.Contains(dis, "PUSH_GLOBAL") {
		t.Errorf("expected PUSH_GLOBAL in disassembly:\n%s", dis)
	}
}

func TestCompileBlock(t *testing.T) {
	chunk, err := compileSource("[ 42 ].")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	dis := chunk.Disassemble("test")
	if !strings.Contains(dis, "CLOSURE") {
		t.Errorf("expected CLOSURE in disassembly:\n%s", dis)
	}
}

func TestCompileBlockWithParams(t *testing.T) {
	chunk, err := compileSource("[ :x | x + 1 ].")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	dis := chunk.Disassemble("test")
	if !strings.Contains(dis, "CLOSURE") {
		t.Errorf("expected CLOSURE in disassembly:\n%s", dis)
	}
}

func TestCompileArrayLiteral(t *testing.T) {
	chunk, err := compileSource("#( 1 2 3 ).")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	dis := chunk.Disassemble("test")
	if !strings.Contains(dis, "MAKE_ARRAY") {
		t.Errorf("expected MAKE_ARRAY in disassembly:\n%s", dis)
	}
}

func TestCompileCascade(t *testing.T) {
	chunk, err := compileSource("Console print: 'a'; print: 'b'.")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	dis := chunk.Disassemble("test")
	// Should have two SEND operations for cascade
	sendCount := strings.Count(dis, "SEND")
	if sendCount < 2 {
		t.Errorf("expected at least 2 SEND operations, got %d:\n%s", sendCount, dis)
	}
	// Should reference both string constants
	if !strings.Contains(dis, "'a'") || !strings.Contains(dis, "'b'") {
		t.Errorf("expected both 'a' and 'b' constants in disassembly:\n%s", dis)
	}
}

func TestCompileIfTrue(t *testing.T) {
	chunk, err := compileSource("true ifTrue: [ 42 ].")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	dis := chunk.Disassemble("test")
	if !strings.Contains(dis, "PUSH_TRUE") {
		t.Errorf("expected PUSH_TRUE in disassembly:\n%s", dis)
	}
	if !strings.Contains(dis, "CLOSURE") {
		t.Errorf("expected CLOSURE in disassembly:\n%s", dis)
	}
	if !strings.Contains(dis, "SEND") {
		t.Errorf("expected SEND in disassembly:\n%s", dis)
	}
}

func TestCompileReturn(t *testing.T) {
	// Return in top-level context
	chunk, err := compileSource("^42.")
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}

	dis := chunk.Disassemble("test")
	if !strings.Contains(dis, "RETURN") {
		t.Errorf("expected RETURN in disassembly:\n%s", dis)
	}
}
