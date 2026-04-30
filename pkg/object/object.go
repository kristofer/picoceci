// Package object defines the runtime value representation for picoceci.
//
// Every picoceci value is represented as an *Object.  Primitive values
// (integers, booleans, nil, characters) are encoded as tagged pointer
// values where possible to avoid heap allocation.
//
// This package is intentionally low-level; higher-level operations are
// in pkg/eval (interpreter) and pkg/runtime (built-in objects).
package object

import "fmt"

// Kind classifies the runtime kind of a picoceci value.
type Kind uint8

const (
	KindNil       Kind = iota
	KindBool           // true / false
	KindSmallInt       // tagged integer
	KindFloat          // IEEE-754 double
	KindChar           // Unicode code point
	KindString         // immutable UTF-8 string
	KindSymbol         // interned string
	KindByteArray      // mutable byte slice
	KindArray          // heterogeneous array
	KindBlock          // closure
	KindObject         // user-defined object
	KindNativeFunc     // Go-implemented callable
)

// MethodDef holds the definition of a picoceci method.
type MethodDef struct {
	Selector string
	Params   []string // parameter names
	Locals   []string // local variable names
	Body     interface{} // []ast.Node (to avoid import cycle; cast by eval)
	Native   func(self *Object, args []*Object) (*Object, error) // for built-ins
}

// Object is the universal value container for the picoceci runtime.
type Object struct {
	Kind Kind

	// Primitive value storage (only one is populated at a time).
	IVal int64   // KindSmallInt
	FVal float64 // KindFloat
	BVal bool    // KindBool
	RVal rune    // KindChar
	SVal string  // KindString, KindSymbol

	// Collection storage.
	Bytes   []byte    // KindByteArray
	Items   []*Object // KindArray

	// Object / block storage.
	Slots   map[string]*Object // instance variable slots
	Methods map[string]*MethodDef
	Env     interface{} // *eval.Env — set by eval package (avoid import cycle)

	// Block-specific.
	Params []string
	Locals []string
	Body   interface{} // []ast.Node

	// Reference count (used by the memory package).
	RefCount int32
}

// --- Constructors -----------------------------------------------------------

// Nil is the singleton nil value.
var Nil = &Object{Kind: KindNil}

// True and False are the singleton boolean values.
var True = &Object{Kind: KindBool, BVal: true}
var False = &Object{Kind: KindBool, BVal: false}

// BoolObject returns True or False.
func BoolObject(b bool) *Object {
	if b {
		return True
	}
	return False
}

// IntObject creates an integer object.
func IntObject(v int64) *Object {
	return &Object{Kind: KindSmallInt, IVal: v}
}

// FloatObject creates a float object.
func FloatObject(v float64) *Object {
	return &Object{Kind: KindFloat, FVal: v}
}

// CharObject creates a character object.
func CharObject(r rune) *Object {
	return &Object{Kind: KindChar, RVal: r}
}

// StringObject creates a string object.
func StringObject(s string) *Object {
	return &Object{Kind: KindString, SVal: s}
}

// SymbolObject creates a symbol object.
// In a full implementation, symbols should be interned via a global table.
func SymbolObject(s string) *Object {
	return &Object{Kind: KindSymbol, SVal: s}
}

// ByteArrayObject creates a byte array object.
func ByteArrayObject(b []byte) *Object {
	cp := make([]byte, len(b))
	copy(cp, b)
	return &Object{Kind: KindByteArray, Bytes: cp}
}

// ArrayObject creates an array object of the given size, filled with nil.
func ArrayObject(size int) *Object {
	items := make([]*Object, size)
	for i := range items {
		items[i] = Nil
	}
	return &Object{Kind: KindArray, Items: items}
}

// NewObject creates a new user-defined object with the given methods.
func NewObject(methods map[string]*MethodDef) *Object {
	return &Object{
		Kind:    KindObject,
		Slots:   make(map[string]*Object),
		Methods: methods,
	}
}

// --- Type tests -------------------------------------------------------------

func (o *Object) IsNil() bool   { return o == nil || o.Kind == KindNil }
func (o *Object) IsBool() bool  { return o.Kind == KindBool }
func (o *Object) IsInt() bool   { return o.Kind == KindSmallInt }
func (o *Object) IsFloat() bool { return o.Kind == KindFloat }
func (o *Object) IsString() bool { return o.Kind == KindString }
func (o *Object) IsSymbol() bool { return o.Kind == KindSymbol }

// Truthy returns whether the object is considered true in a boolean context.
// Only `false` and `nil` are falsy.
func (o *Object) Truthy() bool {
	if o == nil || o.Kind == KindNil {
		return false
	}
	if o.Kind == KindBool {
		return o.BVal
	}
	return true
}

// --- Printing ---------------------------------------------------------------

// PrintString returns the Smalltalk printString representation of o.
func (o *Object) PrintString() string {
	if o == nil {
		return "nil"
	}
	switch o.Kind {
	case KindNil:
		return "nil"
	case KindBool:
		if o.BVal {
			return "true"
		}
		return "false"
	case KindSmallInt:
		return fmt.Sprintf("%d", o.IVal)
	case KindFloat:
		return fmt.Sprintf("%g", o.FVal)
	case KindChar:
		return fmt.Sprintf("$%c", o.RVal)
	case KindString:
		return fmt.Sprintf("'%s'", o.SVal)
	case KindSymbol:
		return "#" + o.SVal
	case KindByteArray:
		s := "#["
		for i, b := range o.Bytes {
			if i > 0 {
				s += " "
			}
			s += fmt.Sprintf("%d", b)
		}
		return s + "]"
	case KindArray:
		s := "("
		for i, item := range o.Items {
			if i > 0 {
				s += " "
			}
			s += item.PrintString()
		}
		return s + " )"
	case KindBlock:
		return "a BlockClosure"
	case KindObject:
		if m, ok := o.Methods["printString"]; ok && m.Native != nil {
			res, err := m.Native(o, nil)
			if err == nil && res != nil && res.Kind == KindString {
				return res.SVal
			}
		}
		return "an Object"
	case KindNativeFunc:
		return "a NativeFunction"
	}
	return "???"
}
