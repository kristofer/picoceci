package bytecode

import (
	"testing"

	"github.com/kristofer/picoceci/pkg/lexer"
	"github.com/kristofer/picoceci/pkg/object"
	"github.com/kristofer/picoceci/pkg/parser"
)

func runVM(src string) (*object.Object, error) {
	l := lexer.NewString(src)
	p := parser.New(l)
	prog, err := p.ParseProgram()
	if err != nil {
		return nil, err
	}

	c := NewCompiler()
	chunk, err := c.Compile(prog.Statements)
	if err != nil {
		return nil, err
	}

	vm := NewVM()
	vm.SetBlocks(c.GetBlocks())
	return vm.Run(chunk)
}

func TestVMIntLiteral(t *testing.T) {
	result, err := runVM("42.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 42 {
		t.Errorf("expected 42, got %s", result.PrintString())
	}
}

func TestVMFloatLiteral(t *testing.T) {
	result, err := runVM("3.14.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindFloat || result.FVal != 3.14 {
		t.Errorf("expected 3.14, got %s", result.PrintString())
	}
}

func TestVMStringLiteral(t *testing.T) {
	result, err := runVM("'hello'.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindString || result.SVal != "hello" {
		t.Errorf("expected 'hello', got %s", result.PrintString())
	}
}

func TestVMSymbolLiteral(t *testing.T) {
	result, err := runVM("#hello.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSymbol || result.SVal != "hello" {
		t.Errorf("expected #hello, got %s", result.PrintString())
	}
}

func TestVMBoolTrue(t *testing.T) {
	result, err := runVM("true.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result != object.True {
		t.Errorf("expected true, got %s", result.PrintString())
	}
}

func TestVMBoolFalse(t *testing.T) {
	result, err := runVM("false.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result != object.False {
		t.Errorf("expected false, got %s", result.PrintString())
	}
}

func TestVMNil(t *testing.T) {
	result, err := runVM("nil.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result != object.Nil {
		t.Errorf("expected nil, got %s", result.PrintString())
	}
}

func TestVMIntAddition(t *testing.T) {
	result, err := runVM("3 + 4.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 7 {
		t.Errorf("expected 7, got %s", result.PrintString())
	}
}

func TestVMIntSubtraction(t *testing.T) {
	result, err := runVM("10 - 4.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 6 {
		t.Errorf("expected 6, got %s", result.PrintString())
	}
}

func TestVMIntMultiplication(t *testing.T) {
	result, err := runVM("3 * 4.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 12 {
		t.Errorf("expected 12, got %s", result.PrintString())
	}
}

func TestVMIntDivision(t *testing.T) {
	result, err := runVM("12 / 4.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 3 {
		t.Errorf("expected 3, got %s", result.PrintString())
	}
}

func TestVMComparison(t *testing.T) {
	tests := []struct {
		src  string
		want bool
	}{
		{"3 < 4.", true},
		{"4 < 3.", false},
		{"3 > 4.", false},
		{"4 > 3.", true},
		{"3 = 3.", true},
		{"3 = 4.", false},
		{"3 <= 3.", true},
		{"3 >= 3.", true},
	}

	for _, tt := range tests {
		result, err := runVM(tt.src)
		if err != nil {
			t.Fatalf("VM error for %q: %v", tt.src, err)
		}
		got := result == object.True
		if got != tt.want {
			t.Errorf("%s: expected %v, got %v", tt.src, tt.want, got)
		}
	}
}

func TestVMStringConcat(t *testing.T) {
	result, err := runVM("'hello', ' world'.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindString || result.SVal != "hello world" {
		t.Errorf("expected 'hello world', got %s", result.PrintString())
	}
}

func TestVMStringSize(t *testing.T) {
	result, err := runVM("'hello' size.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 5 {
		t.Errorf("expected 5, got %s", result.PrintString())
	}
}

func TestVMLocalVariable(t *testing.T) {
	result, err := runVM("| x | x := 42. x.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 42 {
		t.Errorf("expected 42, got %s", result.PrintString())
	}
}

func TestVMMultipleLocals(t *testing.T) {
	result, err := runVM("| x y | x := 3. y := 4. x + y.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 7 {
		t.Errorf("expected 7, got %s", result.PrintString())
	}
}

func TestVMIfTrue(t *testing.T) {
	result, err := runVM("true ifTrue: [ 42 ].")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 42 {
		t.Errorf("expected 42, got %s", result.PrintString())
	}
}

func TestVMIfFalse(t *testing.T) {
	result, err := runVM("false ifFalse: [ 42 ].")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 42 {
		t.Errorf("expected 42, got %s", result.PrintString())
	}
}

func TestVMIfTrueIfFalse(t *testing.T) {
	result, err := runVM("true ifTrue: [ 1 ] ifFalse: [ 2 ].")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 1 {
		t.Errorf("expected 1, got %s", result.PrintString())
	}

	result, err = runVM("false ifTrue: [ 1 ] ifFalse: [ 2 ].")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 2 {
		t.Errorf("expected 2, got %s", result.PrintString())
	}
}

func TestVMBlockValue(t *testing.T) {
	result, err := runVM("[ 42 ] value.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 42 {
		t.Errorf("expected 42, got %s", result.PrintString())
	}
}

func TestVMBlockWithArg(t *testing.T) {
	result, err := runVM("[ :x | x + 1 ] value: 41.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 42 {
		t.Errorf("expected 42, got %s", result.PrintString())
	}
}

func TestVMTimesRepeat(t *testing.T) {
	result, err := runVM("| sum | sum := 0. 3 timesRepeat: [ sum := sum + 1 ]. sum.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 3 {
		t.Errorf("expected 3, got %s", result.PrintString())
	}
}

func TestVMToDo(t *testing.T) {
	result, err := runVM("| sum | sum := 0. 1 to: 3 do: [ :i | sum := sum + i ]. sum.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 6 {
		t.Errorf("expected 6, got %s", result.PrintString())
	}
}

func TestVMArrayLiteral(t *testing.T) {
	result, err := runVM("#( 1 2 3 ).")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindArray {
		t.Fatalf("expected Array, got %s", result.PrintString())
	}
	if len(result.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(result.Items))
	}
}

func TestVMArrayAt(t *testing.T) {
	result, err := runVM("#( 10 20 30 ) at: 2.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 20 {
		t.Errorf("expected 20, got %s", result.PrintString())
	}
}

func TestVMArraySize(t *testing.T) {
	result, err := runVM("#( 1 2 3 4 5 ) size.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 5 {
		t.Errorf("expected 5, got %s", result.PrintString())
	}
}

func TestVMArrayCollect(t *testing.T) {
	result, err := runVM("#( 1 2 3 ) collect: [ :x | x * 2 ].")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindArray {
		t.Fatalf("expected Array, got %s", result.PrintString())
	}
	if len(result.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(result.Items))
	}
	// Check values
	expected := []int64{2, 4, 6}
	for i, want := range expected {
		if result.Items[i].Kind != object.KindSmallInt || result.Items[i].IVal != want {
			t.Errorf("item %d: expected %d, got %s", i, want, result.Items[i].PrintString())
		}
	}
}

func TestVMGlobal(t *testing.T) {
	// Console is a global
	result, err := runVM("Console.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindObject {
		t.Errorf("expected Object (Console), got %s", result.PrintString())
	}
}

func TestVMBoolNot(t *testing.T) {
	result, err := runVM("true not.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result != object.False {
		t.Errorf("expected false, got %s", result.PrintString())
	}

	result, err = runVM("false not.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result != object.True {
		t.Errorf("expected true, got %s", result.PrintString())
	}
}

func TestVMIntAbs(t *testing.T) {
	// picoceci doesn't support unary minus, use negated instead
	result, err := runVM("42 negated abs.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 42 {
		t.Errorf("expected 42, got %s", result.PrintString())
	}
}

func TestVMStringReversed(t *testing.T) {
	result, err := runVM("'hello' reversed.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindString || result.SVal != "olleh" {
		t.Errorf("expected 'olleh', got %s", result.PrintString())
	}
}

func TestVMWhileTrue(t *testing.T) {
	result, err := runVM("| x | x := 0. [ x < 3 ] whileTrue: [ x := x + 1 ]. x.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 3 {
		t.Errorf("expected 3, got %s", result.PrintString())
	}
}

func TestVMReturn(t *testing.T) {
	result, err := runVM("^42.")
	if err != nil {
		t.Fatalf("VM error: %v", err)
	}
	if result.Kind != object.KindSmallInt || result.IVal != 42 {
		t.Errorf("expected 42, got %s", result.PrintString())
	}
}
