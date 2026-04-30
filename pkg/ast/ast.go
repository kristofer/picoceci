// Package ast defines the Abstract Syntax Tree node types for picoceci.
//
// Each source construct has a corresponding Node implementation.
// All nodes carry source position information for error reporting.
// See LANGUAGE_SPEC.md §4 and docs/grammar.ebnf for the grammar.
package ast

import "fmt"

// Pos records a source position.
type Pos struct {
	Line int // 1-based
	Col  int // 1-based byte offset within line
}

func (p Pos) String() string { return fmt.Sprintf("L%d:C%d", p.Line, p.Col) }

// Node is the interface implemented by all AST nodes.
type Node interface {
	nodePos() Pos
	nodeString() string
}

// --- Top-level --------------------------------------------------------------

// Program is the root node of a parsed picoceci source file.
type Program struct {
	Pos        Pos
	Statements []Node
}

func (n *Program) nodePos() Pos    { return n.Pos }
func (n *Program) nodeString() string { return fmt.Sprintf("Program(%d stmts)", len(n.Statements)) }

// ImportDecl represents:  import 'path'.
type ImportDecl struct {
	Pos  Pos
	Path string
}

func (n *ImportDecl) nodePos() Pos    { return n.Pos }
func (n *ImportDecl) nodeString() string { return fmt.Sprintf("Import(%q)", n.Path) }

// ObjectDecl represents:  object Name { ... }
type ObjectDecl struct {
	Pos      Pos
	Name     string
	Composes []string     // names of composed objects
	Slots    []string     // instance variable names
	Methods  []*MethodDef
}

func (n *ObjectDecl) nodePos() Pos    { return n.Pos }
func (n *ObjectDecl) nodeString() string { return fmt.Sprintf("Object(%s)", n.Name) }

// MethodDef is a method inside an object declaration.
type MethodDef struct {
	Pos      Pos
	Selector string   // full selector, e.g. "inc", "+", "at:put:"
	Params   []string // parameter names (in order)
	Locals   []string // local variable names
	Body     []Node
}

func (n *MethodDef) nodePos() Pos    { return n.Pos }
func (n *MethodDef) nodeString() string { return fmt.Sprintf("Method(%s)", n.Selector) }

// InterfaceDecl represents:  interface Name { sigs... }
type InterfaceDecl struct {
	Pos  Pos
	Name string
	Sigs []string // method selectors
}

func (n *InterfaceDecl) nodePos() Pos    { return n.Pos }
func (n *InterfaceDecl) nodeString() string { return fmt.Sprintf("Interface(%s)", n.Name) }

// --- Statements -------------------------------------------------------------

// VarDecl represents:  | x y z |
type VarDecl struct {
	Pos   Pos
	Names []string
}

func (n *VarDecl) nodePos() Pos    { return n.Pos }
func (n *VarDecl) nodeString() string { return fmt.Sprintf("VarDecl(%v)", n.Names) }

// Assign represents:  x := expr
type Assign struct {
	Pos   Pos
	Name  string
	Value Node
}

func (n *Assign) nodePos() Pos    { return n.Pos }
func (n *Assign) nodeString() string { return fmt.Sprintf("Assign(%s)", n.Name) }

// Return represents:  ^expr
type Return struct {
	Pos   Pos
	Value Node
}

func (n *Return) nodePos() Pos    { return n.Pos }
func (n *Return) nodeString() string { return "Return" }

// --- Message sends ----------------------------------------------------------

// Cascade represents:  recv msg1 ; msg2 ; msg3
type Cascade struct {
	Pos      Pos
	Receiver Node
	Messages []Node // UnaryMsg | BinaryMsg | KeywordMsg
}

func (n *Cascade) nodePos() Pos    { return n.Pos }
func (n *Cascade) nodeString() string { return "Cascade" }

// UnaryMsg represents:  receiver selector
type UnaryMsg struct {
	Pos      Pos
	Receiver Node
	Selector string
}

func (n *UnaryMsg) nodePos() Pos    { return n.Pos }
func (n *UnaryMsg) nodeString() string { return fmt.Sprintf("Unary(%s)", n.Selector) }

// BinaryMsg represents:  receiver op arg
type BinaryMsg struct {
	Pos      Pos
	Receiver Node
	Op       string
	Arg      Node
}

func (n *BinaryMsg) nodePos() Pos    { return n.Pos }
func (n *BinaryMsg) nodeString() string { return fmt.Sprintf("Binary(%s)", n.Op) }

// KeywordMsg represents:  receiver key1: arg1 key2: arg2
type KeywordMsg struct {
	Pos      Pos
	Receiver Node
	Keywords []string
	Args     []Node
}

func (n *KeywordMsg) nodePos() Pos    { return n.Pos }
func (n *KeywordMsg) nodeString() string {
	sel := ""
	for _, k := range n.Keywords {
		sel += k
	}
	return fmt.Sprintf("Keyword(%s)", sel)
}

// Selector returns the full keyword selector, e.g. "at:put:".
func (n *KeywordMsg) Selector() string {
	s := ""
	for _, k := range n.Keywords {
		s += k
	}
	return s
}

// --- Literals ---------------------------------------------------------------

// IntLit is an integer literal.
type IntLit struct {
	Pos   Pos
	Value int64
	Raw   string // original source text
}

func (n *IntLit) nodePos() Pos    { return n.Pos }
func (n *IntLit) nodeString() string { return fmt.Sprintf("Int(%d)", n.Value) }

// FloatLit is a floating-point literal.
type FloatLit struct {
	Pos   Pos
	Value float64
	Raw   string
}

func (n *FloatLit) nodePos() Pos    { return n.Pos }
func (n *FloatLit) nodeString() string { return fmt.Sprintf("Float(%g)", n.Value) }

// StringLit is a string literal (unquoted content).
type StringLit struct {
	Pos   Pos
	Value string
}

func (n *StringLit) nodePos() Pos    { return n.Pos }
func (n *StringLit) nodeString() string { return fmt.Sprintf("String(%q)", n.Value) }

// SymbolLit is a symbol literal (without leading #).
type SymbolLit struct {
	Pos   Pos
	Value string
}

func (n *SymbolLit) nodePos() Pos    { return n.Pos }
func (n *SymbolLit) nodeString() string { return fmt.Sprintf("Symbol(#%s)", n.Value) }

// CharLit is a character literal ($A).
type CharLit struct {
	Pos   Pos
	Value rune
}

func (n *CharLit) nodePos() Pos    { return n.Pos }
func (n *CharLit) nodeString() string { return fmt.Sprintf("Char($%c)", n.Value) }

// BoolLit is true or false.
type BoolLit struct {
	Pos   Pos
	Value bool
}

func (n *BoolLit) nodePos() Pos    { return n.Pos }
func (n *BoolLit) nodeString() string { return fmt.Sprintf("Bool(%v)", n.Value) }

// NilLit is nil.
type NilLit struct {
	Pos Pos
}

func (n *NilLit) nodePos() Pos    { return n.Pos }
func (n *NilLit) nodeString() string { return "Nil" }

// ArrayLit is a literal array: #( 1 'two' #three )
type ArrayLit struct {
	Pos      Pos
	Elements []Node
}

func (n *ArrayLit) nodePos() Pos    { return n.Pos }
func (n *ArrayLit) nodeString() string { return fmt.Sprintf("Array(%d)", len(n.Elements)) }

// ByteArrayLit is a literal byte array: #[ 1 2 3 ]
type ByteArrayLit struct {
	Pos   Pos
	Bytes []byte
}

func (n *ByteArrayLit) nodePos() Pos    { return n.Pos }
func (n *ByteArrayLit) nodeString() string { return fmt.Sprintf("ByteArray(%d bytes)", len(n.Bytes)) }

// --- Identifiers and blocks -------------------------------------------------

// Ident is a variable reference.
type Ident struct {
	Pos  Pos
	Name string
}

func (n *Ident) nodePos() Pos    { return n.Pos }
func (n *Ident) nodeString() string { return fmt.Sprintf("Ident(%s)", n.Name) }

// SelfExpr is the `self` pseudo-variable.
type SelfExpr struct{ Pos Pos }

func (n *SelfExpr) nodePos() Pos    { return n.Pos }
func (n *SelfExpr) nodeString() string { return "self" }

// SuperExpr is the `super` pseudo-variable.
type SuperExpr struct{ Pos Pos }

func (n *SuperExpr) nodePos() Pos    { return n.Pos }
func (n *SuperExpr) nodeString() string { return "super" }

// Block represents a block closure: [ :p | body ]
type Block struct {
	Pos    Pos
	Params []string
	Locals []string
	Body   []Node
}

func (n *Block) nodePos() Pos    { return n.Pos }
func (n *Block) nodeString() string {
	return fmt.Sprintf("Block(params=%v, body=%d stmts)", n.Params, len(n.Body))
}
