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
	obj := evalSrc(t, "| x: Any | x := 42. x.")
	if obj.IVal != 42 {
		t.Errorf("assignment: got %v", obj.PrintString())
	}
}

func TestEval_MultipleAssignments(t *testing.T) {
	obj := evalSrc(t, "| x: Int  y: Int | x := 3. y := 4. x + y.")
	if obj.IVal != 7 {
		t.Errorf("x+y: got %v", obj.PrintString())
	}
}

// --- blocks -----------------------------------------------------------------

func TestEval_BlockValue(t *testing.T) {
	obj := evalSrc(t, "| b: Any | b := [ 42 ]. b value.")
	if obj.IVal != 42 {
		t.Errorf("block value: got %v", obj.PrintString())
	}
}

func TestEval_BlockWithArg(t *testing.T) {
	obj := evalSrc(t, "| b: Any | b := [ :x | x + 1 ]. b value: 5.")
	if obj.IVal != 6 {
		t.Errorf("block value:: got %v", obj.PrintString())
	}
}

func TestEval_BlockClosure(t *testing.T) {
	src := `
| adder: Any  result: Int |
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
| x: Int |
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
| x: Int |
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
    | count: Int |
    init  [ count := 0 ]
    inc   [ count := count + 1. ^self ]
    value [ ^count ]
}
| c: Counter |
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
    | count: Int |
    init  [ count := 0 ]
    inc   [ count := count + 1. ^self ]
    value [ ^count ]
}
| c: Counter |
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

// --- composition ------------------------------------------------------------

func TestEval_Composition_SlotsInherited(t *testing.T) {
	src := `
object Base {
    | x: Int |
    init  [ x := 10 ]
    getX  [ ^x ]
}
object Derived {
    compose Base.
}
| d: Derived |
d := Derived new.
d getX.`
	obj := evalSrc(t, src)
	if obj.IVal != 10 {
		t.Errorf("composed slot: got %v, want 10", obj.PrintString())
	}
}

func TestEval_Composition_MethodInherited(t *testing.T) {
	src := `
object Counter {
    | count: Int |
    init  [ count := 0 ]
    inc   [ count := count + 1. ^self ]
    value [ ^count ]
}
object LoggedCounter {
    compose Counter.
    inc [
        count := count + 1.
        ^self
    ]
}
| c: LoggedCounter |
c := LoggedCounter new.
c inc.
c inc.
c value.`
	obj := evalSrc(t, src)
	if obj.IVal != 2 {
		t.Errorf("composition method: got %v, want 2", obj.PrintString())
	}
}

func TestEval_Composition_SuperDispatch(t *testing.T) {
	src := `
object Base {
    | x: Int |
    init  [ x := 0 ]
    inc   [ x := x + 1. ^self ]
    value [ ^x ]
}
object Child {
    compose Base.
    inc [
        super inc.
        super inc.
        ^self
    ]
}
| c: Child |
c := Child new.
c inc.
c value.`
	obj := evalSrc(t, src)
	if obj.IVal != 2 {
		t.Errorf("super dispatch: got %v, want 2", obj.PrintString())
	}
}

func TestEval_Composition_GlobalAccessFromMethod(t *testing.T) {
	// Methods must be able to access global variables like Counter.
	src := `
object Foo {
    | n: Int |
    init  [ n := 0 ]
    run   [ n := n + 1. ^n ]
}
object Bar {
    compose Foo.
    run [
        | result: Int |
        result := super run.
        ^result * 2
    ]
}
| b: Bar |
b := Bar new.
b run.`
	obj := evalSrc(t, src)
	if obj.IVal != 2 {
		t.Errorf("global access in method: got %v, want 2", obj.PrintString())
	}
}

// --- missing number methods -------------------------------------------------

func TestEval_IntSqrt(t *testing.T) {
	obj := evalSrc(t, "9 sqrt.")
	if obj.Kind != object.KindFloat || obj.FVal != 3.0 {
		t.Errorf("9 sqrt: got %v", obj.PrintString())
	}
}

func TestEval_FloatFloor(t *testing.T) {
	obj := evalSrc(t, "3.7 floor.")
	if obj.Kind != object.KindSmallInt || obj.IVal != 3 {
		t.Errorf("3.7 floor: got %v", obj.PrintString())
	}
}

func TestEval_FloatCeiling(t *testing.T) {
	obj := evalSrc(t, "3.2 ceiling.")
	if obj.Kind != object.KindSmallInt || obj.IVal != 4 {
		t.Errorf("3.2 ceiling: got %v", obj.PrintString())
	}
}

func TestEval_FloatRounded(t *testing.T) {
	obj := evalSrc(t, "3.5 rounded.")
	if obj.Kind != object.KindSmallInt || obj.IVal != 4 {
		t.Errorf("3.5 rounded: got %v", obj.PrintString())
	}
}

func TestEval_IntFloor(t *testing.T) {
	obj := evalSrc(t, "5 floor.")
	if obj.Kind != object.KindSmallInt || obj.IVal != 5 {
		t.Errorf("5 floor: got %v", obj.PrintString())
	}
}

// --- missing string methods -------------------------------------------------

func TestEval_StringAt(t *testing.T) {
	obj := evalSrc(t, "'hello' at: 1.")
	if obj.Kind != object.KindChar || obj.RVal != 'h' {
		t.Errorf("'hello' at: 1: got %v", obj.PrintString())
	}
}

func TestEval_StringAt_OutOfBounds(t *testing.T) {
	err := evalErr(t, "'hello' at: 10.")
	if !strings.Contains(err.Error(), "IndexOutOfBounds") {
		t.Errorf("expected IndexOutOfBounds, got %v", err)
	}
}

func TestEval_StringCopyFromTo(t *testing.T) {
	obj := evalSrc(t, "'hello world' copyFrom: 7 to: 11.")
	if obj.Kind != object.KindString || obj.SVal != "world" {
		t.Errorf("copyFrom:to:: got %q", obj.SVal)
	}
}

func TestEval_StringIncludesSubString(t *testing.T) {
	obj := evalSrc(t, "'hello world' includesSubString: 'world'.")
	if obj != object.True {
		t.Errorf("includesSubString: got %v", obj.PrintString())
	}
}

func TestEval_StringAsInteger(t *testing.T) {
	obj := evalSrc(t, "'42' asInteger.")
	if obj.Kind != object.KindSmallInt || obj.IVal != 42 {
		t.Errorf("'42' asInteger: got %v", obj.PrintString())
	}
}

func TestEval_StringAsFloat(t *testing.T) {
	obj := evalSrc(t, "'3.14' asFloat.")
	if obj.Kind != object.KindFloat || obj.FVal != 3.14 {
		t.Errorf("'3.14' asFloat: got %v", obj.PrintString())
	}
}

// --- Array class ------------------------------------------------------------

func TestEval_ArrayNewSize(t *testing.T) {
	obj := evalSrc(t, "Array new: 3.")
	if obj.Kind != object.KindArray || len(obj.Items) != 3 {
		t.Errorf("Array new: 3: got %v", obj.PrintString())
	}
}

func TestEval_ArrayNewWithAll(t *testing.T) {
	obj := evalSrc(t, "Array new: 3 withAll: 0.")
	if obj.Kind != object.KindArray || len(obj.Items) != 3 {
		t.Fatalf("Array new:withAll:: got %v", obj.PrintString())
	}
	for _, item := range obj.Items {
		if item.Kind != object.KindSmallInt || item.IVal != 0 {
			t.Errorf("expected 0 element, got %v", item.PrintString())
		}
	}
}

func TestEval_ArrayDetect(t *testing.T) {
	obj := evalSrc(t, "#(1 2 3 4 5) detect: [ :each | each > 3 ].")
	if obj.Kind != object.KindSmallInt || obj.IVal != 4 {
		t.Errorf("detect: got %v, want 4", obj.PrintString())
	}
}

func TestEval_ArrayDetect_NotFound(t *testing.T) {
	err := evalErr(t, "#(1 2 3) detect: [ :each | each > 10 ].")
	if !strings.Contains(err.Error(), "ElementNotFound") {
		t.Errorf("expected ElementNotFound, got %v", err)
	}
}

// --- to:do: loop ------------------------------------------------------------

func TestEval_ToDo(t *testing.T) {
	src := `
| sum: Int |
sum := 0.
1 to: 5 do: [ :i | sum := sum + i ].
sum.`
	obj := evalSrc(t, src)
	if obj.IVal != 15 {
		t.Errorf("to:do: got %v, want 15", obj.PrintString())
	}
}

// --- typed variables --------------------------------------------------------

func TestEval_TypedVarDecl_ZeroValues(t *testing.T) {
// Int zero value is 0
obj := evalSrc(t, "| x: Int | x.")
if obj.Kind != object.KindSmallInt || obj.IVal != 0 {
t.Errorf("Int zero value: got %v, want 0", obj.PrintString())
}
}

func TestEval_TypedVarDecl_FloatZero(t *testing.T) {
obj := evalSrc(t, "| x: Float | x.")
if obj.Kind != object.KindFloat || obj.FVal != 0.0 {
t.Errorf("Float zero value: got %v, want 0.0", obj.PrintString())
}
}

func TestEval_TypedVarDecl_BoolZero(t *testing.T) {
obj := evalSrc(t, "| x: Bool | x.")
if obj != object.False {
t.Errorf("Bool zero value: got %v, want false", obj.PrintString())
}
}

func TestEval_TypedVarDecl_StringZero(t *testing.T) {
obj := evalSrc(t, "| x: String | x.")
if obj.Kind != object.KindString || obj.SVal != "" {
t.Errorf("String zero value: got %v, want empty string", obj.PrintString())
}
}

func TestEval_TypedVarDecl_AnyIsNil(t *testing.T) {
obj := evalSrc(t, "| x: Any | x.")
if !obj.IsNil() {
t.Errorf("Any zero value: got %v, want nil", obj.PrintString())
}
}

func TestEval_TypedVar_TypeCheckPasses(t *testing.T) {
// Assigning correct type should work fine
obj := evalSrc(t, "| x: Int | x := 42. x.")
if obj.Kind != object.KindSmallInt || obj.IVal != 42 {
t.Errorf("typed assignment: got %v, want 42", obj.PrintString())
}
}

func TestEval_TypedVar_TypeCheckFails(t *testing.T) {
// Assigning wrong type should raise TypeError
err := evalErr(t, "| x: Int | x := 'hello'.")
if !strings.Contains(err.Error(), "TypeError") {
t.Errorf("expected TypeError, got %v", err)
}
}

func TestEval_TypedVar_AnyAllowsAnyType(t *testing.T) {
// Any-typed vars accept any value
obj := evalSrc(t, "| x: Any | x := 'hello'. x.")
if obj.Kind != object.KindString || obj.SVal != "hello" {
t.Errorf("Any typed var: got %v, want 'hello'", obj.PrintString())
}
}

func TestEval_TypedSlot_TypeCheckPasses(t *testing.T) {
src := `
object Box {
    | val: Int |
    init  [ val := 0 ]
    set: v [ val := v ]
    get    [ ^val ]
}
| b: Box |
b := Box new.
b set: 99.
b get.`
obj := evalSrc(t, src)
if obj.Kind != object.KindSmallInt || obj.IVal != 99 {
t.Errorf("typed slot: got %v, want 99", obj.PrintString())
}
}

func TestEval_TypedSlot_TypeCheckFails(t *testing.T) {
src := `
object Box {
    | val: Int |
    init  [ val := 0 ]
    set: v [ val := v ]
}
| b: Box |
b := Box new.
b set: 'oops'.`
err := evalErr(t, src)
if !strings.Contains(err.Error(), "TypeError") {
t.Errorf("expected TypeError for slot type mismatch, got %v", err)
}
}

func TestEval_TypedSlot_ZeroValue(t *testing.T) {
// Float slot zero value is 0.0 (no explicit init needed)
src := `
object Sensor {
    | temp: Float |
    reading [ ^temp ]
}
| s: Sensor |
s := Sensor new.
s reading.`
obj := evalSrc(t, src)
if obj.Kind != object.KindFloat || obj.FVal != 0.0 {
t.Errorf("Float slot zero value: got %v, want 0.0", obj.PrintString())
}
}
