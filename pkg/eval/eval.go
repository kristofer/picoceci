// Package eval implements the picoceci tree-walking interpreter.
//
// The evaluator walks an AST produced by pkg/parser and executes it,
// using the runtime objects defined in pkg/object and pkg/runtime.
//
// Phase 2 deliverable — see IMPLEMENTATION_PLAN.md.
package eval

import (
	"fmt"

	"github.com/kristofer/picoceci/pkg/ast"
	"github.com/kristofer/picoceci/pkg/object"
)

// Error represents a picoceci runtime error.
type Error struct {
	Kind    string
	Message string
	Pos     ast.Pos
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s (at %s)", e.Kind, e.Message, e.Pos)
}

// Env is a lexical scope mapping variable names to objects.
// When selfObj is non-nil, slot names in selfObj.Slots are visible as variables.
type Env struct {
	vars            map[string]*object.Object
	outer           *Env
	selfObj         *object.Object // non-nil in method environments
	composedMethods map[string]*object.MethodDef // for super dispatch in method envs
}

// NewEnv creates a fresh top-level environment.
func NewEnv() *Env {
	return &Env{vars: make(map[string]*object.Object)}
}

// child creates a nested scope.
func (e *Env) child() *Env {
	return &Env{vars: make(map[string]*object.Object), outer: e}
}

// getComposedMethods walks the scope chain to find the nearest composedMethods table.
func (e *Env) getComposedMethods() map[string]*object.MethodDef {
	if e == nil {
		return nil
	}
	if e.composedMethods != nil {
		return e.composedMethods
	}
	return e.outer.getComposedMethods()
}

// Get looks up a variable, walking the scope chain.
// If a selfObj is set at this level, slot names are visible.
func (e *Env) Get(name string) (*object.Object, bool) {
	if v, ok := e.vars[name]; ok {
		return v, true
	}
	// Instance slot lookup.
	if e.selfObj != nil && e.selfObj.Slots != nil {
		if v, ok := e.selfObj.Slots[name]; ok {
			return v, true
		}
	}
	if e.outer != nil {
		return e.outer.Get(name)
	}
	return nil, false
}

// Set creates or updates a variable in the innermost scope where it exists,
// or defines it in the current scope if not found anywhere.
func (e *Env) Set(name string, val *object.Object) {
	if _, ok := e.vars[name]; ok {
		e.vars[name] = val
		return
	}
	// Write to instance slot if this is a method env with a selfObj.
	if e.selfObj != nil && e.selfObj.Slots != nil {
		if _, ok := e.selfObj.Slots[name]; ok {
			e.selfObj.Slots[name] = val
			return
		}
	}
	if e.outer != nil {
		if e.outer.setExisting(name, val) {
			return
		}
	}
	e.vars[name] = val
}

func (e *Env) setExisting(name string, val *object.Object) bool {
	if _, ok := e.vars[name]; ok {
		e.vars[name] = val
		return true
	}
	if e.selfObj != nil && e.selfObj.Slots != nil {
		if _, ok := e.selfObj.Slots[name]; ok {
			e.selfObj.Slots[name] = val
			return true
		}
	}
	if e.outer != nil {
		return e.outer.setExisting(name, val)
	}
	return false
}

// Define declares a new variable in the current (not outer) scope.
func (e *Env) Define(name string) {
	e.vars[name] = object.Nil
}


// Interpreter holds the interpreter state.
type Interpreter struct {
	globals         *Env
	objectTemplates map[string]*ast.ObjectDecl // AST templates for compose support
}

// New creates a new Interpreter with built-in objects registered.
func New() *Interpreter {
	interp := &Interpreter{
		globals:         NewEnv(),
		objectTemplates: make(map[string]*ast.ObjectDecl),
	}
	registerBuiltins(interp.globals)
	return interp
}

// Eval evaluates a list of AST statements in the global environment.
func (interp *Interpreter) Eval(nodes []ast.Node) (*object.Object, error) {
	var result *object.Object = object.Nil
	for _, n := range nodes {
		val, err := interp.evalNode(n, interp.globals)
		if err != nil {
			return nil, err
		}
		result = val
	}
	return result, nil
}

// evalNode dispatches on node type.
func (interp *Interpreter) evalNode(n ast.Node, env *Env) (*object.Object, error) {
	switch node := n.(type) {
	case *ast.Program:
		return interp.evalStatements(node.Statements, env)
	case *ast.VarDecl:
		for _, name := range node.Names {
			env.Define(name)
		}
		return object.Nil, nil
	case *ast.Assign:
		val, err := interp.evalNode(node.Value, env)
		if err != nil {
			return nil, err
		}
		env.Set(node.Name, val)
		return val, nil
	case *ast.Return:
		val, err := interp.evalNode(node.Value, env)
		if err != nil {
			return nil, err
		}
		return nil, &returnSignal{value: val}
	case *ast.NilLit:
		return object.Nil, nil
	case *ast.BoolLit:
		return object.BoolObject(node.Value), nil
	case *ast.IntLit:
		return object.IntObject(node.Value), nil
	case *ast.FloatLit:
		return object.FloatObject(node.Value), nil
	case *ast.StringLit:
		return object.StringObject(node.Value), nil
	case *ast.SymbolLit:
		return object.SymbolObject(node.Value), nil
	case *ast.CharLit:
		return object.CharObject(node.Value), nil
	case *ast.ByteArrayLit:
		return object.ByteArrayObject(node.Bytes), nil
	case *ast.ArrayLit:
		arr := object.ArrayObject(len(node.Elements))
		for i, elem := range node.Elements {
			v, err := interp.evalNode(elem, env)
			if err != nil {
				return nil, err
			}
			arr.Items[i] = v
		}
		return arr, nil
	case *ast.Ident:
		if v, ok := env.Get(node.Name); ok {
			return v, nil
		}
		return nil, &Error{Kind: "UndefinedVariable", Message: "undefined: " + node.Name, Pos: node.Pos}
	case *ast.SelfExpr:
		if v, ok := env.Get("self"); ok {
			return v, nil
		}
		return object.Nil, nil
	case *ast.SuperExpr:
		if v, ok := env.Get("self"); ok {
			return v, nil
		}
		return object.Nil, nil
	case *ast.ThisContextExpr:
		// thisContext is not fully implemented; return nil for now.
		return object.Nil, nil
	case *ast.AnonObjectLit:
		inst := &object.Object{
			Kind:    object.KindObject,
			Slots:   make(map[string]*object.Object),
			Methods: make(map[string]*object.MethodDef),
		}
		for _, slot := range node.Slots {
			val, err := interp.evalNode(slot.Value, env)
			if err != nil {
				return nil, err
			}
			inst.Slots[slot.Name] = val
			// Expose each slot as a unary accessor method.  Use an
			// immediately-applied wrapper so each closure captures its
			// own copy of the slot name rather than a shared variable.
			name := slot.Name
			inst.Methods[name] = &object.MethodDef{
				Selector: name,
				Native: func(n string) func(*object.Object, []*object.Object) (*object.Object, error) {
					return func(self *object.Object, _ []*object.Object) (*object.Object, error) {
						if v, ok := self.Slots[n]; ok {
							return v, nil
						}
						return object.Nil, nil
					}
				}(name),
			}
		}
		return inst, nil
	case *ast.Block:
		blk := &object.Object{
			Kind:   object.KindBlock,
			Params: node.Params,
			Locals: node.Locals,
			Body:   node.Body,
			Env:    env,
		}
		return blk, nil
	case *ast.UnaryMsg:
		if _, ok := node.Receiver.(*ast.SuperExpr); ok {
			self, _ := env.Get("self")
			if self == nil {
				self = object.Nil
			}
			return interp.superSend(self, node.Selector, nil, node.Pos, env)
		}
		recv, err := interp.evalNode(node.Receiver, env)
		if err != nil {
			return nil, err
		}
		return interp.send(recv, node.Selector, nil, node.Pos)
	case *ast.BinaryMsg:
		if _, ok := node.Receiver.(*ast.SuperExpr); ok {
			self, _ := env.Get("self")
			if self == nil {
				self = object.Nil
			}
			arg, err := interp.evalNode(node.Arg, env)
			if err != nil {
				return nil, err
			}
			return interp.superSend(self, node.Op, []*object.Object{arg}, node.Pos, env)
		}
		recv, err := interp.evalNode(node.Receiver, env)
		if err != nil {
			return nil, err
		}
		arg, err := interp.evalNode(node.Arg, env)
		if err != nil {
			return nil, err
		}
		return interp.send(recv, node.Op, []*object.Object{arg}, node.Pos)
	case *ast.KeywordMsg:
		if _, ok := node.Receiver.(*ast.SuperExpr); ok {
			self, _ := env.Get("self")
			if self == nil {
				self = object.Nil
			}
			args := make([]*object.Object, len(node.Args))
			var err error
			for i, a := range node.Args {
				args[i], err = interp.evalNode(a, env)
				if err != nil {
					return nil, err
				}
			}
			sel := ""
			for _, k := range node.Keywords {
				sel += k
			}
			return interp.superSend(self, sel, args, node.Pos, env)
		}
		recv, err := interp.evalNode(node.Receiver, env)
		if err != nil {
			return nil, err
		}
		args := make([]*object.Object, len(node.Args))
		for i, a := range node.Args {
			args[i], err = interp.evalNode(a, env)
			if err != nil {
				return nil, err
			}
		}
		sel := ""
		for _, k := range node.Keywords {
			sel += k
		}
		return interp.send(recv, sel, args, node.Pos)
	case *ast.Cascade:
		recv, err := interp.evalNode(node.Receiver, env)
		if err != nil {
			return nil, err
		}
		var last *object.Object = recv
		for _, msg := range node.Messages {
			switch m := msg.(type) {
			case *ast.UnaryMsg:
				last, err = interp.send(recv, m.Selector, nil, m.Pos)
			case *ast.BinaryMsg:
				arg, e := interp.evalNode(m.Arg, env)
				if e != nil {
					return nil, e
				}
				last, err = interp.send(recv, m.Op, []*object.Object{arg}, m.Pos)
			case *ast.KeywordMsg:
				args := make([]*object.Object, len(m.Args))
				for i, a := range m.Args {
					args[i], err = interp.evalNode(a, env)
					if err != nil {
						return nil, err
					}
				}
				sel := ""
				for _, k := range m.Keywords {
					sel += k
				}
				last, err = interp.send(recv, sel, args, m.Pos)
			}
			if err != nil {
				return nil, err
			}
		}
		return last, nil
	case *ast.ObjectDecl:
		interp.registerObjectDecl(node, env)
		return object.Nil, nil
	case *ast.InterfaceDecl:
		// Interface declarations are recorded for runtime checking (future).
		return object.Nil, nil
	case *ast.ImportDecl:
		// Module loading — Phase 4.
		return object.Nil, nil
	default:
		return nil, fmt.Errorf("eval: unhandled node type %T", n)
	}
}

func (interp *Interpreter) evalStatements(nodes []ast.Node, env *Env) (*object.Object, error) {
	var result *object.Object = object.Nil
	for _, n := range nodes {
		val, err := interp.evalNode(n, env)
		if err != nil {
			if rs, ok := err.(*returnSignal); ok {
				return rs.value, nil
			}
			return nil, err
		}
		result = val
	}
	return result, nil
}

// send performs a message send: recv <selector> args.
func (interp *Interpreter) send(recv *object.Object, selector string, args []*object.Object, p ast.Pos) (*object.Object, error) {
	if recv == nil {
		recv = object.Nil
	}

	// Look up method in receiver's methods map.
	if recv.Methods != nil {
		if m, ok := recv.Methods[selector]; ok {
			return interp.applyMethod(recv, m, args, p)
		}
	}

	// Built-in dispatch.
	result, err, handled := builtinDispatch(interp, recv, selector, args, p)
	if handled {
		return result, err
	}

	// MessageNotUnderstood
	return nil, &Error{
		Kind:    "MessageNotUnderstood",
		Message: fmt.Sprintf("%s does not understand #%s", kindDescription(recv), selector),
		Pos:     p,
	}
}

func (interp *Interpreter) applyMethod(self *object.Object, m *object.MethodDef, args []*object.Object, p ast.Pos) (*object.Object, error) {
	if m.Native != nil {
		return m.Native(self, args)
	}
	// AST body
	body, ok := m.Body.([]ast.Node)
	if !ok {
		return object.Nil, nil
	}
	methodEnv := &Env{
		vars:            make(map[string]*object.Object),
		selfObj:         self,
		composedMethods: self.ComposedMethods,
		outer:           interp.globals,
	}
	methodEnv.Define("self")
	methodEnv.vars["self"] = self // bypass slot lookup for "self" itself
	for i, param := range m.Params {
		methodEnv.Define(param)
		if i < len(args) {
			methodEnv.vars[param] = args[i]
		}
	}
	for _, local := range m.Locals {
		methodEnv.Define(local)
	}
	return interp.evalStatements(body, methodEnv)
}

// CallBlock invokes a block object with the given arguments.
func (interp *Interpreter) CallBlock(blk *object.Object, args []*object.Object) (*object.Object, error) {
	if blk.Kind != object.KindBlock {
		return nil, fmt.Errorf("not a block")
	}
	body, ok := blk.Body.([]ast.Node)
	if !ok {
		return object.Nil, nil
	}
	outer, _ := blk.Env.(*Env)
	blockEnv := &Env{vars: make(map[string]*object.Object), outer: outer}
	for i, param := range blk.Params {
		blockEnv.Define(param)
		if i < len(args) {
			blockEnv.Set(param, args[i])
		}
	}
	for _, local := range blk.Locals {
		blockEnv.Define(local)
	}
	return interp.evalStatements(body, blockEnv)
}

// superSend dispatches a message to the composed (super) methods of an object.
func (interp *Interpreter) superSend(self *object.Object, selector string, args []*object.Object, p ast.Pos, env *Env) (*object.Object, error) {
	composed := env.getComposedMethods()
	if composed != nil {
		if m, ok := composed[selector]; ok {
			return interp.applyMethod(self, m, args, p)
		}
	}
	return nil, &Error{
		Kind:    "MessageNotUnderstood",
		Message: fmt.Sprintf("%s does not understand super #%s", kindDescription(self), selector),
		Pos:     p,
	}
}

// registerObjectDecl registers an object template so that `Name new` works.
func (interp *Interpreter) registerObjectDecl(decl *ast.ObjectDecl, env *Env) {
	// Save template for composition lookup.
	interp.objectTemplates[decl.Name] = decl

	// Collect slots and methods from all composed objects (in declaration order).
	allSlots := make([]string, 0)
	composedMethods := make(map[string]*object.MethodDef)

	for _, composeName := range decl.Composes {
		if composeDecl, ok := interp.objectTemplates[composeName]; ok {
			allSlots = append(allSlots, composeDecl.Slots...)
		}
		if composeFactory, ok := env.Get(composeName); ok {
			for sel, m := range composeFactory.Methods {
				if sel != "new" {
					composedMethods[sel] = m
				}
			}
		}
	}

	// Own slots come after composed slots.
	allSlots = append(allSlots, decl.Slots...)

	// Build the factory's method table: composed methods first, then own
	// methods override them.
	factory := &object.Object{
		Kind:            object.KindObject,
		Slots:           make(map[string]*object.Object),
		Methods:         make(map[string]*object.MethodDef),
		ComposedMethods: composedMethods,
	}

	for sel, m := range composedMethods {
		factory.Methods[sel] = m
	}

	for _, mdef := range decl.Methods {
		mdef := mdef // capture
		factory.Methods[mdef.Selector] = &object.MethodDef{
			Selector: mdef.Selector,
			Params:   mdef.Params,
			Locals:   mdef.Locals,
			Body:     mdef.Body,
		}
	}

	// `new` method: create an instance and call `init` if it exists.
	capturedSlots := allSlots
	factory.Methods["new"] = &object.MethodDef{
		Selector: "new",
		Native: func(self *object.Object, _ []*object.Object) (*object.Object, error) {
			inst := &object.Object{
				Kind:            object.KindObject,
				Slots:           make(map[string]*object.Object),
				Methods:         self.Methods, // share method table
				ComposedMethods: self.ComposedMethods,
			}
			for _, slot := range capturedSlots {
				inst.Slots[slot] = object.Nil
			}
			// Call init if defined.
			if m, ok := inst.Methods["init"]; ok {
				_, err := interp.applyMethod(inst, m, nil, ast.Pos{})
				if err != nil {
					return nil, err
				}
			}
			return inst, nil
		},
	}

	env.Define(decl.Name)
	env.Set(decl.Name, factory)
}

// kindDescription returns a short human-readable description for error messages.
func kindDescription(o *object.Object) string {
	if o == nil || o.Kind == object.KindNil {
		return "nil"
	}
	switch o.Kind {
	case object.KindBool:
		if o.BVal {
			return "true"
		}
		return "false"
	case object.KindSmallInt:
		return fmt.Sprintf("Integer(%d)", o.IVal)
	case object.KindFloat:
		return fmt.Sprintf("Float(%g)", o.FVal)
	case object.KindString:
		return "a String"
	case object.KindSymbol:
		return "a Symbol"
	case object.KindArray:
		return "an Array"
	case object.KindBlock:
		return "a BlockClosure"
	case object.KindObject:
		return "an Object"
	}
	return "an Object"
}

// returnSignal is used to implement non-local return from methods.
type returnSignal struct{ value *object.Object }

func (r *returnSignal) Error() string { return "return" }
