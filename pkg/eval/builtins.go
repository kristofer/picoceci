package eval

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/kristofer/picoceci/pkg/ast"
	"github.com/kristofer/picoceci/pkg/object"
)

// registerBuiltins populates the global environment with built-in objects.
func registerBuiltins(env *Env) {
	env.Define("nil")
	env.Set("nil", object.Nil)
	env.Define("true")
	env.Set("true", object.True)
	env.Define("false")
	env.Set("false", object.False)

	// Console / Transcript
	console := makeConsole()
	env.Define("Console")
	env.Set("Console", console)
	env.Define("Transcript")
	env.Set("Transcript", console)

	// Array class object — responds to new: and new:withAll:
	arrayClass := makeArrayClass()
	env.Define("Array")
	env.Set("Array", arrayClass)
}

func displayString(o *object.Object) string {
	if o == nil {
		return "nil"
	}
	// For strings and symbols, return raw value (no quotes) — like displayString.
	if o.Kind == object.KindString {
		return o.SVal
	}
	if o.Kind == object.KindSymbol {
		return o.SVal
	}
	return o.PrintString()
}

func makeConsole() *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
	}
	o.Methods["print:"] = &object.MethodDef{Native: func(_ *object.Object, args []*object.Object) (*object.Object, error) {
		if len(args) > 0 {
			fmt.Print(displayString(args[0]))
		}
		return object.Nil, nil
	}}
	o.Methods["println:"] = &object.MethodDef{Native: func(_ *object.Object, args []*object.Object) (*object.Object, error) {
		if len(args) > 0 {
			fmt.Println(displayString(args[0]))
		}
		return object.Nil, nil
	}}
	o.Methods["show:"] = o.Methods["println:"]
	o.Methods["nl"] = &object.MethodDef{Native: func(_ *object.Object, _ []*object.Object) (*object.Object, error) {
		fmt.Println()
		return object.Nil, nil
	}}
	return o
}

func makeArrayClass() *object.Object {
	o := &object.Object{
		Kind:    object.KindObject,
		Slots:   make(map[string]*object.Object),
		Methods: make(map[string]*object.MethodDef),
	}
	o.Methods["new:"] = &object.MethodDef{Native: func(_ *object.Object, args []*object.Object) (*object.Object, error) {
		if len(args) > 0 && args[0].Kind == object.KindSmallInt {
			return object.ArrayObject(int(args[0].IVal)), nil
		}
		return object.ArrayObject(0), nil
	}}
	o.Methods["new:withAll:"] = &object.MethodDef{Native: func(_ *object.Object, args []*object.Object) (*object.Object, error) {
		if len(args) == 2 && args[0].Kind == object.KindSmallInt {
			size := int(args[0].IVal)
			arr := &object.Object{Kind: object.KindArray, Items: make([]*object.Object, size)}
			for i := range arr.Items {
				arr.Items[i] = args[1]
			}
			return arr, nil
		}
		return object.ArrayObject(0), nil
	}}
	return o
}

// builtinDispatch handles message sends to primitive types.
// Returns (result, error, handled).
func builtinDispatch(interp *Interpreter, recv *object.Object, sel string, args []*object.Object, p ast.Pos) (*object.Object, error, bool) {
	switch recv.Kind {
	case object.KindSmallInt:
		return intDispatch(interp, recv, sel, args, p)
	case object.KindFloat:
		return floatDispatch(recv, sel, args, p)
	case object.KindBool:
		return boolDispatch(interp, recv, sel, args, p)
	case object.KindString:
		return stringDispatch(interp, recv, sel, args, p)
	case object.KindSymbol:
		return symbolDispatch(recv, sel, args, p)
	case object.KindArray:
		return arrayDispatch(interp, recv, sel, args, p)
	case object.KindBlock:
		return blockDispatch(interp, recv, sel, args, p)
	case object.KindNil:
		return nilDispatch(recv, sel, args, p)
	}
	return nil, nil, false
}

// --- nil --------------------------------------------------------------------

func nilDispatch(recv *object.Object, sel string, args []*object.Object, p ast.Pos) (*object.Object, error, bool) {
	switch sel {
	case "isNil":
		return object.True, nil, true
	case "notNil":
		return object.False, nil, true
	case "printString":
		return object.StringObject("nil"), nil, true
	case "=":
		if len(args) > 0 && args[0].Kind == object.KindNil {
			return object.True, nil, true
		}
		return object.False, nil, true
	case "~=":
		if len(args) > 0 && args[0].Kind == object.KindNil {
			return object.False, nil, true
		}
		return object.True, nil, true
	}
	return nil, nil, false
}

// --- integers ---------------------------------------------------------------

func intDispatch(interp *Interpreter, recv *object.Object, sel string, args []*object.Object, p ast.Pos) (*object.Object, error, bool) {
	a := recv.IVal
	arg0 := func() (*object.Object, bool) {
		if len(args) == 0 {
			return nil, false
		}
		return args[0], true
	}
	numArg := func() (int64, float64, bool, bool) {
		o, ok := arg0()
		if !ok {
			return 0, 0, false, false
		}
		if o.Kind == object.KindSmallInt {
			return o.IVal, 0, true, false
		}
		if o.Kind == object.KindFloat {
			return 0, o.FVal, false, true
		}
		return 0, 0, false, false
	}

	switch sel {
	case "+":
		iv, fv, isInt, isFloat := numArg()
		if isInt {
			return object.IntObject(a + iv), nil, true
		}
		if isFloat {
			return object.FloatObject(float64(a) + fv), nil, true
		}
	case "-":
		iv, fv, isInt, isFloat := numArg()
		if isInt {
			return object.IntObject(a - iv), nil, true
		}
		if isFloat {
			return object.FloatObject(float64(a) - fv), nil, true
		}
	case "*":
		iv, fv, isInt, isFloat := numArg()
		if isInt {
			return object.IntObject(a * iv), nil, true
		}
		if isFloat {
			return object.FloatObject(float64(a) * fv), nil, true
		}
	case "/":
		iv, fv, isInt, isFloat := numArg()
		if isInt {
			if iv == 0 {
				return nil, &Error{Kind: "ZeroDivision", Message: "division by zero", Pos: p}, true
			}
			if a%iv == 0 {
				return object.IntObject(a / iv), nil, true
			}
			return object.FloatObject(float64(a) / float64(iv)), nil, true
		}
		if isFloat {
			return object.FloatObject(float64(a) / fv), nil, true
		}
	case "//":
		iv, _, isInt, _ := numArg()
		if isInt {
			if iv == 0 {
				return nil, &Error{Kind: "ZeroDivision", Message: "division by zero", Pos: p}, true
			}
			return object.IntObject(a / iv), nil, true
		}
	case "\\\\":
		iv, _, isInt, _ := numArg()
		if isInt {
			if iv == 0 {
				return nil, &Error{Kind: "ZeroDivision", Message: "modulo by zero", Pos: p}, true
			}
			return object.IntObject(a % iv), nil, true
		}
	case "=":
		o, ok := arg0()
		if ok {
			if o.Kind == object.KindSmallInt {
				return object.BoolObject(a == o.IVal), nil, true
			}
			if o.Kind == object.KindFloat {
				return object.BoolObject(float64(a) == o.FVal), nil, true
			}
			return object.False, nil, true
		}
	case "~=":
		o, ok := arg0()
		if ok {
			if o.Kind == object.KindSmallInt {
				return object.BoolObject(a != o.IVal), nil, true
			}
			return object.True, nil, true
		}
	case "<":
		iv, fv, isInt, isFloat := numArg()
		if isInt {
			return object.BoolObject(a < iv), nil, true
		}
		if isFloat {
			return object.BoolObject(float64(a) < fv), nil, true
		}
	case ">":
		iv, fv, isInt, isFloat := numArg()
		if isInt {
			return object.BoolObject(a > iv), nil, true
		}
		if isFloat {
			return object.BoolObject(float64(a) > fv), nil, true
		}
	case "<=":
		iv, fv, isInt, isFloat := numArg()
		if isInt {
			return object.BoolObject(a <= iv), nil, true
		}
		if isFloat {
			return object.BoolObject(float64(a) <= fv), nil, true
		}
	case ">=":
		iv, fv, isInt, isFloat := numArg()
		if isInt {
			return object.BoolObject(a >= iv), nil, true
		}
		if isFloat {
			return object.BoolObject(float64(a) >= fv), nil, true
		}
	case "abs":
		if a < 0 {
			return object.IntObject(-a), nil, true
		}
		return recv, nil, true
	case "negated":
		return object.IntObject(-a), nil, true
	case "sqrt":
		return object.FloatObject(math.Sqrt(float64(a))), nil, true
	case "floor":
		return recv, nil, true
	case "ceiling":
		return recv, nil, true
	case "rounded":
		return recv, nil, true
	case "asFloat":
		return object.FloatObject(float64(a)), nil, true
	case "asInteger":
		return recv, nil, true
	case "printString":
		return object.StringObject(fmt.Sprintf("%d", a)), nil, true
	case "isNil":
		return object.False, nil, true
	case "notNil":
		return object.True, nil, true
	case "timesRepeat:":
		blk, ok := arg0()
		if ok && blk.Kind == object.KindBlock {
			for i := int64(0); i < a; i++ {
				if _, err := interp.CallBlock(blk, nil); err != nil {
					return nil, err, true
				}
			}
			return recv, nil, true
		}
	case "to:do:":
		if len(args) == 2 {
			limit := args[0]
			blk := args[1]
			if limit.Kind == object.KindSmallInt && blk.Kind == object.KindBlock {
				for i := a; i <= limit.IVal; i++ {
					if _, err := interp.CallBlock(blk, []*object.Object{object.IntObject(i)}); err != nil {
						return nil, err, true
					}
				}
				return recv, nil, true
			}
		}
	case "to:":
		// Returns a range-like — for now return self (placeholder)
		return recv, nil, true
	}
	return nil, nil, false
}

// --- floats -----------------------------------------------------------------

func floatDispatch(recv *object.Object, sel string, args []*object.Object, p ast.Pos) (*object.Object, error, bool) {
	a := recv.FVal
	numArg := func() (float64, bool) {
		if len(args) == 0 {
			return 0, false
		}
		o := args[0]
		if o.Kind == object.KindFloat {
			return o.FVal, true
		}
		if o.Kind == object.KindSmallInt {
			return float64(o.IVal), true
		}
		return 0, false
	}

	switch sel {
	case "+":
		b, ok := numArg()
		if ok {
			return object.FloatObject(a + b), nil, true
		}
	case "-":
		b, ok := numArg()
		if ok {
			return object.FloatObject(a - b), nil, true
		}
	case "*":
		b, ok := numArg()
		if ok {
			return object.FloatObject(a * b), nil, true
		}
	case "/":
		b, ok := numArg()
		if ok {
			if b == 0 {
				return nil, &Error{Kind: "ZeroDivision", Message: "division by zero", Pos: p}, true
			}
			return object.FloatObject(a / b), nil, true
		}
	case "=":
		b, ok := numArg()
		if ok {
			return object.BoolObject(a == b), nil, true
		}
		return object.False, nil, true
	case "<":
		b, ok := numArg()
		if ok {
			return object.BoolObject(a < b), nil, true
		}
	case ">":
		b, ok := numArg()
		if ok {
			return object.BoolObject(a > b), nil, true
		}
	case "<=":
		b, ok := numArg()
		if ok {
			return object.BoolObject(a <= b), nil, true
		}
	case ">=":
		b, ok := numArg()
		if ok {
			return object.BoolObject(a >= b), nil, true
		}
	case "abs":
		if a < 0 {
			return object.FloatObject(-a), nil, true
		}
		return recv, nil, true
	case "negated":
		return object.FloatObject(-a), nil, true
	case "sqrt":
		return object.FloatObject(math.Sqrt(a)), nil, true
	case "floor":
		return object.IntObject(int64(math.Floor(a))), nil, true
	case "ceiling":
		return object.IntObject(int64(math.Ceil(a))), nil, true
	case "rounded":
		return object.IntObject(int64(math.Round(a))), nil, true
	case "printString":
		return object.StringObject(fmt.Sprintf("%g", a)), nil, true
	case "asFloat":
		return recv, nil, true
	case "asInteger":
		return object.IntObject(int64(a)), nil, true
	case "isNil":
		return object.False, nil, true
	case "notNil":
		return object.True, nil, true
	}
	return nil, nil, false
}

// --- booleans ---------------------------------------------------------------

func boolDispatch(interp *Interpreter, recv *object.Object, sel string, args []*object.Object, p ast.Pos) (*object.Object, error, bool) {
	b := recv.BVal
	switch sel {
	case "ifTrue:":
		if b && len(args) > 0 && args[0].Kind == object.KindBlock {
			res, err := interp.CallBlock(args[0], nil)
			return res, err, true
		}
		return object.Nil, nil, true
	case "ifFalse:":
		if !b && len(args) > 0 && args[0].Kind == object.KindBlock {
			res, err := interp.CallBlock(args[0], nil)
			return res, err, true
		}
		return object.Nil, nil, true
	case "ifTrue:ifFalse:":
		if len(args) == 2 {
			blk := args[0]
			if !b {
				blk = args[1]
			}
			if blk.Kind == object.KindBlock {
				res, err := interp.CallBlock(blk, nil)
				return res, err, true
			}
		}
		return object.Nil, nil, true
	case "ifFalse:ifTrue:":
		if len(args) == 2 {
			blk := args[1]
			if !b {
				blk = args[0]
			}
			if blk.Kind == object.KindBlock {
				res, err := interp.CallBlock(blk, nil)
				return res, err, true
			}
		}
		return object.Nil, nil, true
	case "not":
		return object.BoolObject(!b), nil, true
	case "&":
		if len(args) > 0 {
			return object.BoolObject(b && args[0].Truthy()), nil, true
		}
	case "|":
		if len(args) > 0 {
			return object.BoolObject(b || args[0].Truthy()), nil, true
		}
	case "=":
		if len(args) > 0 && args[0].Kind == object.KindBool {
			return object.BoolObject(b == args[0].BVal), nil, true
		}
		return object.False, nil, true
	case "printString":
		if b {
			return object.StringObject("true"), nil, true
		}
		return object.StringObject("false"), nil, true
	case "isNil":
		return object.False, nil, true
	case "notNil":
		return object.True, nil, true
	}
	return nil, nil, false
}

// --- strings ----------------------------------------------------------------

func stringDispatch(interp *Interpreter, recv *object.Object, sel string, args []*object.Object, p ast.Pos) (*object.Object, error, bool) {
	s := recv.SVal
	switch sel {
	case "size":
		return object.IntObject(int64(len([]rune(s)))), nil, true
	case "printString":
		return object.StringObject("'" + strings.ReplaceAll(s, "'", "''") + "'"), nil, true
	case "displayString":
		return object.StringObject(s), nil, true
	case ",":
		if len(args) > 0 {
			other := args[0].SVal
			return object.StringObject(s + other), nil, true
		}
	case "reversed":
		runes := []rune(s)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		return object.StringObject(string(runes)), nil, true
	case "asUppercase":
		return object.StringObject(strings.ToUpper(s)), nil, true
	case "asLowercase":
		return object.StringObject(strings.ToLower(s)), nil, true
	case "asSymbol":
		return object.SymbolObject(s), nil, true
	case "asBytes":
		return object.ByteArrayObject([]byte(s)), nil, true
	case "=":
		if len(args) > 0 {
			if args[0].Kind == object.KindString {
				return object.BoolObject(s == args[0].SVal), nil, true
			}
			return object.False, nil, true
		}
	case "~=":
		if len(args) > 0 {
			if args[0].Kind == object.KindString {
				return object.BoolObject(s != args[0].SVal), nil, true
			}
			return object.True, nil, true
		}
	case "<":
		if len(args) > 0 && args[0].Kind == object.KindString {
			return object.BoolObject(s < args[0].SVal), nil, true
		}
	case ">":
		if len(args) > 0 && args[0].Kind == object.KindString {
			return object.BoolObject(s > args[0].SVal), nil, true
		}
	case "trimSeparators":
		return object.StringObject(strings.TrimSpace(s)), nil, true
	case "isNil":
		return object.False, nil, true
	case "notNil":
		return object.True, nil, true
	case "at:":
		if len(args) > 0 && args[0].Kind == object.KindSmallInt {
			runes := []rune(s)
			idx64 := args[0].IVal
			if idx64 < 1 || idx64 > int64(len(runes)) {
				return nil, &Error{Kind: "IndexOutOfBounds", Message: fmt.Sprintf("index %d out of bounds (size %d)", idx64, len(runes)), Pos: p}, true
			}
			return object.CharObject(runes[int(idx64)-1]), nil, true
		}
	case "copyFrom:to:":
		if len(args) == 2 && args[0].Kind == object.KindSmallInt && args[1].Kind == object.KindSmallInt {
			runes := []rune(s)
			n := int64(len(runes))
			start64 := args[0].IVal - 1
			stop64 := args[1].IVal
			if start64 < 0 {
				start64 = 0
			}
			if stop64 > n {
				stop64 = n
			}
			if start64 > stop64 {
				return object.StringObject(""), nil, true
			}
			return object.StringObject(string(runes[int(start64):int(stop64)])), nil, true
		}
	case "includesSubString:":
		if len(args) > 0 && args[0].Kind == object.KindString {
			return object.BoolObject(strings.Contains(s, args[0].SVal)), nil, true
		}
	case "asInteger":
		if v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil {
			return object.IntObject(v), nil, true
		}
		return object.Nil, nil, true
	case "asFloat":
		if v, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
			return object.FloatObject(v), nil, true
		}
		return object.Nil, nil, true
	case "do:":
		if len(args) > 0 && args[0].Kind == object.KindBlock {
			for _, r := range s {
				if _, err := interp.CallBlock(args[0], []*object.Object{object.CharObject(r)}); err != nil {
					return nil, err, true
				}
			}
			return recv, nil, true
		}
	}
	return nil, nil, false
}

// --- symbols ----------------------------------------------------------------

func symbolDispatch(recv *object.Object, sel string, args []*object.Object, p ast.Pos) (*object.Object, error, bool) {
	switch sel {
	case "asString":
		return object.StringObject(recv.SVal), nil, true
	case "printString":
		return object.StringObject("#" + recv.SVal), nil, true
	case "=":
		if len(args) > 0 && args[0].Kind == object.KindSymbol {
			return object.BoolObject(recv.SVal == args[0].SVal), nil, true
		}
		return object.False, nil, true
	case "asSymbol":
		return recv, nil, true
	case "isNil":
		return object.False, nil, true
	case "notNil":
		return object.True, nil, true
	}
	return nil, nil, false
}

// --- arrays -----------------------------------------------------------------

func arrayDispatch(interp *Interpreter, recv *object.Object, sel string, args []*object.Object, p ast.Pos) (*object.Object, error, bool) {
	items := recv.Items
	switch sel {
	case "size":
		return object.IntObject(int64(len(items))), nil, true
	case "at:":
		if len(args) > 0 && args[0].Kind == object.KindSmallInt {
			idx64 := args[0].IVal
			if idx64 < 1 || idx64 > int64(len(items)) {
				return nil, &Error{Kind: "IndexOutOfBounds", Message: fmt.Sprintf("index %d out of bounds (size %d)", idx64, len(items)), Pos: p}, true
			}
			return items[int(idx64)-1], nil, true
		}
	case "at:put:":
		if len(args) == 2 && args[0].Kind == object.KindSmallInt {
			idx64 := args[0].IVal
			if idx64 < 1 || idx64 > int64(len(items)) {
				return nil, &Error{Kind: "IndexOutOfBounds", Message: fmt.Sprintf("index %d out of bounds", idx64), Pos: p}, true
			}
			items[int(idx64)-1] = args[1]
			return args[1], nil, true
		}
	case "first":
		if len(items) > 0 {
			return items[0], nil, true
		}
		return object.Nil, nil, true
	case "last":
		if len(items) > 0 {
			return items[len(items)-1], nil, true
		}
		return object.Nil, nil, true
	case "do:":
		if len(args) > 0 && args[0].Kind == object.KindBlock {
			for _, item := range items {
				if _, err := interp.CallBlock(args[0], []*object.Object{item}); err != nil {
					return nil, err, true
				}
			}
			return recv, nil, true
		}
	case "collect:":
		if len(args) > 0 && args[0].Kind == object.KindBlock {
			result := object.ArrayObject(len(items))
			for i, item := range items {
				v, err := interp.CallBlock(args[0], []*object.Object{item})
				if err != nil {
					return nil, err, true
				}
				result.Items[i] = v
			}
			return result, nil, true
		}
	case "select:":
		if len(args) > 0 && args[0].Kind == object.KindBlock {
			var result []*object.Object
			for _, item := range items {
				v, err := interp.CallBlock(args[0], []*object.Object{item})
				if err != nil {
					return nil, err, true
				}
				if v.Truthy() {
					result = append(result, item)
				}
			}
			arr := &object.Object{Kind: object.KindArray, Items: result}
			return arr, nil, true
		}
	case "inject:into:":
		if len(args) == 2 && args[1].Kind == object.KindBlock {
			acc := args[0]
			for _, item := range items {
				v, err := interp.CallBlock(args[1], []*object.Object{acc, item})
				if err != nil {
					return nil, err, true
				}
				acc = v
			}
			return acc, nil, true
		}
	case "detect:":
		if len(args) > 0 && args[0].Kind == object.KindBlock {
			for _, item := range items {
				v, err := interp.CallBlock(args[0], []*object.Object{item})
				if err != nil {
					return nil, err, true
				}
				if v.Truthy() {
					return item, nil, true
				}
			}
			return nil, &Error{Kind: "ElementNotFound", Message: "detect: no element satisfies the block", Pos: p}, true
		}
	case "withIndexDo:":
		if len(args) > 0 && args[0].Kind == object.KindBlock {
			for i, item := range items {
				if _, err := interp.CallBlock(args[0], []*object.Object{object.IntObject(int64(i + 1)), item}); err != nil {
					return nil, err, true
				}
			}
			return recv, nil, true
		}
	case "printString":
		var parts []string
		for _, item := range items {
			parts = append(parts, item.PrintString())
		}
		return object.StringObject("(" + strings.Join(parts, " ") + " )"), nil, true
	case "isNil":
		return object.False, nil, true
	case "notNil":
		return object.True, nil, true
	}
	return nil, nil, false
}

// --- blocks -----------------------------------------------------------------

func blockDispatch(interp *Interpreter, recv *object.Object, sel string, args []*object.Object, p ast.Pos) (*object.Object, error, bool) {
	switch sel {
	case "value":
		res, err := interp.CallBlock(recv, nil)
		return res, err, true
	case "value:":
		res, err := interp.CallBlock(recv, args)
		return res, err, true
	case "value:value:":
		res, err := interp.CallBlock(recv, args)
		return res, err, true
	case "valueWithArguments:":
		if len(args) > 0 && args[0].Kind == object.KindArray {
			res, err := interp.CallBlock(recv, args[0].Items)
			return res, err, true
		}
	case "whileTrue:":
		if len(args) > 0 && args[0].Kind == object.KindBlock {
			for {
				cond, err := interp.CallBlock(recv, nil)
				if err != nil {
					return nil, err, true
				}
				if !cond.Truthy() {
					break
				}
				if _, err = interp.CallBlock(args[0], nil); err != nil {
					return nil, err, true
				}
			}
			return object.Nil, nil, true
		}
	case "whileTrue":
		for {
			cond, err := interp.CallBlock(recv, nil)
			if err != nil {
				return nil, err, true
			}
			if !cond.Truthy() {
				break
			}
		}
		return object.Nil, nil, true
	case "whileFalse:":
		if len(args) > 0 && args[0].Kind == object.KindBlock {
			for {
				cond, err := interp.CallBlock(recv, nil)
				if err != nil {
					return nil, err, true
				}
				if cond.Truthy() {
					break
				}
				if _, err = interp.CallBlock(args[0], nil); err != nil {
					return nil, err, true
				}
			}
			return object.Nil, nil, true
		}
	case "on:do:":
		if len(args) == 2 {
			_, err := interp.CallBlock(recv, nil)
			if err != nil {
				// Check if it's a picoceci Error — if so, call the handler block.
				if _, ok := err.(*Error); ok {
					errObj := object.StringObject(err.Error())
					res, herr := interp.CallBlock(args[1], []*object.Object{errObj})
					return res, herr, true
				}
				return nil, err, true
			}
			return object.Nil, nil, true
		}
	case "ensure:":
		if len(args) > 0 && args[0].Kind == object.KindBlock {
			res, err := interp.CallBlock(recv, nil)
			_, _ = interp.CallBlock(args[0], nil) // always run
			return res, err, true
		}
	case "printString":
		return object.StringObject("a BlockClosure"), nil, true
	case "isNil":
		return object.False, nil, true
	case "notNil":
		return object.True, nil, true
	}
	return nil, nil, false
}

// expose for use in blockDispatch above (same package)
// (actual interpreter instance is threaded through all dispatch functions)
