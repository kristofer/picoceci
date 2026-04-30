package eval_test

import (
	"strings"
	"testing"

	"github.com/kristofer/picoceci/pkg/eval"
	"github.com/kristofer/picoceci/pkg/lexer"
	"github.com/kristofer/picoceci/pkg/object"
	"github.com/kristofer/picoceci/pkg/parser"
)

// evalSrc parses and evaluates a picoceci snippet, returning the last value.
func evalSrc(t *testing.T, src string) *object.Object {
	t.Helper()
	l := lexer.NewString(src)
	p := parser.New(l)
	prog, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	interp := eval.New()
	result, err := interp.Eval(prog.Statements)
	if err != nil {
		t.Fatalf("eval error: %v", err)
	}
	return result
}

// evalErr expects an error and returns it.
func evalErr(t *testing.T, src string) error {
	t.Helper()
	l := lexer.NewString(src)
	p := parser.New(l)
	prog, _ := p.ParseProgram()
	interp := eval.New()
	_, err := interp.Eval(prog.Statements)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	return err
}

// --- literals ---------------------------------------------------------------

func TestEval_IntegerLiteral(t *testing.T) {
	obj := evalSrc(t, "42.")
	if obj.Kind != object.KindSmallInt || obj.IVal != 42 {
		t.Errorf("got %v, want SmallInt(42)", obj.PrintString())
	}
}

func TestEval_FloatLiteral(t *testing.T) {
	obj := evalSrc(t, "3.14.")
	if obj.Kind != object.KindFloat || obj.FVal != 3.14 {
		t.Errorf("got %v", obj.PrintString())
	}
}

func TestEval_StringLiteral(t *testing.T) {
	obj := evalSrc(t, "'hello'.")
	if obj.Kind != object.KindString || obj.SVal != "hello" {
		t.Errorf("got %v", obj.PrintString())
	}
}

func TestEval_SymbolLiteral(t *testing.T) {
	obj := evalSrc(t, "#foo.")
	if obj.Kind != object.KindSymbol || obj.SVal != "foo" {
		t.Errorf("got %v", obj.PrintString())
	}
}

func TestEval_BoolTrue(t *testing.T) {
	obj := evalSrc(t, "true.")
	if obj != object.True {
		t.Errorf("got %v, want true", obj.PrintString())
	}
}

func TestEval_BoolFalse(t *testing.T) {
	obj := evalSrc(t, "false.")
	if obj != object.False {
		t.Errorf("got %v, want false", obj.PrintString())
	}
}

func TestEval_Nil(t *testing.T) {
	obj := evalSrc(t, "nil.")
	if !obj.IsNil() {
		t.Errorf("got %v, want nil", obj.PrintString())
	}
}

// --- arithmetic -------------------------------------------------------------

func TestEval_IntAddition(t *testing.T) {
	obj := evalSrc(t, "3 + 4.")
	if obj.Kind != object.KindSmallInt || obj.IVal != 7 {
		t.Errorf("3+4: got %v", obj.PrintString())
	}
}

func TestEval_IntSubtraction(t *testing.T) {
	obj := evalSrc(t, "10 - 3.")
	if obj.IVal != 7 {
		t.Errorf("10-3: got %v", obj.PrintString())
	}
}

func TestEval_IntMultiplication(t *testing.T) {
	obj := evalSrc(t, "6 * 7.")
	if obj.IVal != 42 {
		t.Errorf("6*7: got %v", obj.PrintString())
	}
}

func TestEval_IntDivision_Exact(t *testing.T) {
	obj := evalSrc(t, "10 / 2.")
	if obj.Kind != object.KindSmallInt || obj.IVal != 5 {
		t.Errorf("10/2: got %v", obj.PrintString())
	}
}

func TestEval_IntDivision_Float(t *testing.T) {
	obj := evalSrc(t, "7 / 2.")
	if obj.Kind != object.KindFloat || obj.FVal != 3.5 {
		t.Errorf("7/2: got %v", obj.PrintString())
	}
}

func TestEval_IntFloorDivision(t *testing.T) {
	obj := evalSrc(t, "7 // 2.")
	if obj.IVal != 3 {
		t.Errorf("7//2: got %v", obj.PrintString())
	}
}

func TestEval_IntModulo(t *testing.T) {
	obj := evalSrc(t, "7 \\\\ 3.")
	if obj.IVal != 1 {
		t.Errorf("7 mod 3: got %v", obj.PrintString())
	}
}

func TestEval_DivisionByZero(t *testing.T) {
	err := evalErr(t, "1 / 0.")
	if !strings.Contains(err.Error(), "ZeroDivision") {
		t.Errorf("expected ZeroDivision, got %v", err)
	}
}

// --- comparison -------------------------------------------------------------

func TestEval_IntEqual(t *testing.T) {
	obj := evalSrc(t, "3 = 3.")
	if obj != object.True {
		t.Errorf("3=3: got %v", obj.PrintString())
	}
}

func TestEval_IntLess(t *testing.T) {
	obj := evalSrc(t, "2 < 5.")
	if obj != object.True {
		t.Errorf("2<5: got %v", obj.PrintString())
	}
}

func TestEval_IntGreater(t *testing.T) {
	obj := evalSrc(t, "5 > 2.")
	if obj != object.True {
		t.Errorf("5>2: got %v", obj.PrintString())
	}
}

// --- string messages --------------------------------------------------------

func TestEval_StringSize(t *testing.T) {
	obj := evalSrc(t, "'hello' size.")
	if obj.IVal != 5 {
		t.Errorf("'hello' size: got %v", obj.PrintString())
	}
}

func TestEval_StringConcat(t *testing.T) {
	obj := evalSrc(t, "'hello' , ' world'.")
	if obj.SVal != "hello world" {
		t.Errorf("concat: got %q", obj.SVal)
	}
}

func TestEval_StringReversed(t *testing.T) {
	obj := evalSrc(t, "'hello' reversed.")
	if obj.SVal != "olleh" {
		t.Errorf("reversed: got %q", obj.SVal)
	}
}

func TestEval_StringPrintString(t *testing.T) {
	obj := evalSrc(t, "'hi' printString.")
	if obj.SVal != "'hi'" {
		t.Errorf("printString: got %q", obj.SVal)
	}
}

// --- booleans ---------------------------------------------------------------

func TestEval_IfTrue(t *testing.T) {
	obj := evalSrc(t, "true ifTrue: [ 99 ].")
	if obj.IVal != 99 {
		t.Errorf("ifTrue: got %v", obj.PrintString())
	}
}

func TestEval_IfFalse(t *testing.T) {
	obj := evalSrc(t, "false ifFalse: [ 77 ].")
	if obj.IVal != 77 {
		t.Errorf("ifFalse: got %v", obj.PrintString())
	}
}

func TestEval_IfTrueIfFalse(t *testing.T) {
	obj := evalSrc(t, "(3 > 2) ifTrue: [ 1 ] ifFalse: [ 2 ].")
	if obj.IVal != 1 {
		t.Errorf("ifTrue:ifFalse:: got %v", obj.PrintString())
	}
}

func TestEval_BoolNot(t *testing.T) {
	obj := evalSrc(t, "true not.")
	if obj != object.False {
		t.Errorf("true not: got %v", obj.PrintString())
	}
}

// --- assignment and variable ------------------------------------------------

func TestEval_Assignment(t *testing.T) {
	obj := evalSrc(t, "| x | x := 42. x.")
	if obj.IVal != 42 {
		t.Errorf("assignment: got %v", obj.PrintString())
	}
}

func TestEval_MultipleAssignments(t *testing.T) {
	obj := evalSrc(t, "| x y | x := 3. y := 4. x + y.")
	if obj.IVal != 7 {
		t.Errorf("x+y: got %v", obj.PrintString())
	}
}

// --- blocks -----------------------------------------------------------------

func TestEval_BlockValue(t *testing.T) {
	obj := evalSrc(t, "| b | b := [ 42 ]. b value.")
	if obj.IVal != 42 {
		t.Errorf("block value: got %v", obj.PrintString())
	}
}

func TestEval_BlockWithArg(t *testing.T) {
	obj := evalSrc(t, "| b | b := [ :x | x + 1 ]. b value: 5.")
	if obj.IVal != 6 {
		t.Errorf("block value:: got %v", obj.PrintString())
	}
}

func TestEval_BlockClosure(t *testing.T) {
	src := `
| adder result |
adder := [ :n | [ :x | x + n ] ].
result := (adder value: 5) value: 3.
result.`
	obj := evalSrc(t, src)
	if obj.IVal != 8 {
		t.Errorf("closure: got %v", obj.PrintString())
	}
}

func TestEval_WhileTrue(t *testing.T) {
	src := `
| x |
x := 0.
[ x < 5 ] whileTrue: [ x := x + 1 ].
x.`
	obj := evalSrc(t, src)
	if obj.IVal != 5 {
		t.Errorf("whileTrue: got %v", obj.PrintString())
	}
}

func TestEval_TimesRepeat(t *testing.T) {
	src := `
| x |
x := 0.
5 timesRepeat: [ x := x + 1 ].
x.`
	obj := evalSrc(t, src)
	if obj.IVal != 5 {
		t.Errorf("timesRepeat: got %v", obj.PrintString())
	}
}

// --- arrays -----------------------------------------------------------------

func TestEval_ArrayAt(t *testing.T) {
	obj := evalSrc(t, "#(10 20 30) at: 2.")
	if obj.IVal != 20 {
		t.Errorf("at:2: got %v", obj.PrintString())
	}
}

func TestEval_ArrayCollect(t *testing.T) {
	obj := evalSrc(t, "#(1 2 3) collect: [ :each | each * 2 ].")
	if obj.Kind != object.KindArray || len(obj.Items) != 3 {
		t.Fatalf("collect: got %v", obj.PrintString())
	}
	if obj.Items[0].IVal != 2 || obj.Items[1].IVal != 4 || obj.Items[2].IVal != 6 {
		t.Errorf("collect values: %v %v %v", obj.Items[0].IVal, obj.Items[1].IVal, obj.Items[2].IVal)
	}
}

func TestEval_ArraySelect(t *testing.T) {
	obj := evalSrc(t, "#(1 2 3 4) select: [ :each | each > 2 ].")
	if obj.Kind != object.KindArray || len(obj.Items) != 2 {
		t.Fatalf("select: got %v (len %d)", obj.PrintString(), len(obj.Items))
	}
}

func TestEval_ArrayInjectInto(t *testing.T) {
	obj := evalSrc(t, "#(1 2 3 4 5) inject: 0 into: [ :acc :each | acc + each ].")
	if obj.IVal != 15 {
		t.Errorf("inject:into: got %v", obj.PrintString())
	}
}

func TestEval_ArrayIndexOutOfBounds(t *testing.T) {
	err := evalErr(t, "#(1 2 3) at: 10.")
	if !strings.Contains(err.Error(), "IndexOutOfBounds") {
		t.Errorf("expected IndexOutOfBounds, got %v", err)
	}
}

// --- object declarations ----------------------------------------------------

func TestEval_ObjectDecl_New(t *testing.T) {
	src := `
object Counter {
    | count |
    init  [ count := 0 ]
    inc   [ count := count + 1. ^self ]
    value [ ^count ]
}
| c |
c := Counter new.
c value.`
	obj := evalSrc(t, src)
	if obj.IVal != 0 {
		t.Errorf("initial value: got %v, want 0", obj.PrintString())
	}
}

func TestEval_ObjectDecl_Method(t *testing.T) {
	src := `
object Counter {
    | count |
    init  [ count := 0 ]
    inc   [ count := count + 1. ^self ]
    value [ ^count ]
}
| c |
c := Counter new.
c inc.
c inc.
c inc.
c value.`
	obj := evalSrc(t, src)
	if obj.IVal != 3 {
		t.Errorf("after 3 inc: got %v, want 3", obj.PrintString())
	}
}

// --- error handling ---------------------------------------------------------

func TestEval_MessageNotUnderstood(t *testing.T) {
	err := evalErr(t, "42 unknownMessage.")
	if !strings.Contains(err.Error(), "MessageNotUnderstood") {
		t.Errorf("expected MessageNotUnderstood, got %v", err)
	}
}

func TestEval_UndefinedVariable(t *testing.T) {
	err := evalErr(t, "undeclaredVar.")
	if !strings.Contains(err.Error(), "UndefinedVariable") {
		t.Errorf("expected UndefinedVariable, got %v", err)
	}
}
