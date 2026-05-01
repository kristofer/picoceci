package bytecode

import (
	"fmt"
	"math"

	"github.com/kristofer/picoceci/pkg/ast"
	"github.com/kristofer/picoceci/pkg/object"
)

// Compiler compiles AST nodes to bytecode.
type Compiler struct {
	chunk  *Chunk  // current chunk being compiled
	scope  *scope  // current scope
	blocks []*CompiledBlock // compiled blocks (for closures)

	// For method compilation
	isMethod bool
	selfSlotNames []string // instance variable names when compiling a method
}

// NewCompiler creates a new compiler.
func NewCompiler() *Compiler {
	return &Compiler{
		chunk:  NewChunk(),
		scope:  newScope(nil),
		blocks: make([]*CompiledBlock, 0),
	}
}

// Compile compiles a program (list of statements) into a Chunk.
func (c *Compiler) Compile(nodes []ast.Node) (*Chunk, error) {
	for _, node := range nodes {
		if err := c.compileNode(node); err != nil {
			return nil, err
		}
		// Pop result of expression statements (except last)
	}

	// If chunk is empty, push nil
	if len(c.chunk.Code) == 0 {
		c.emitOp(OpPushNil, 1)
	}

	c.emitOp(OpReturn, 1)
	return c.chunk, nil
}

// CompileExpression compiles a single expression and returns its value.
func (c *Compiler) CompileExpression(node ast.Node) (*Chunk, error) {
	if err := c.compileNode(node); err != nil {
		return nil, err
	}
	c.emitOp(OpReturn, lineOf(node))
	return c.chunk, nil
}

// compileNode dispatches to the appropriate compilation method.
func (c *Compiler) compileNode(node ast.Node) error {
	switch n := node.(type) {
	case *ast.Program:
		return c.compileProgram(n)
	case *ast.NilLit:
		c.emitOp(OpPushNil, n.Pos.Line)
	case *ast.BoolLit:
		if n.Value {
			c.emitOp(OpPushTrue, n.Pos.Line)
		} else {
			c.emitOp(OpPushFalse, n.Pos.Line)
		}
	case *ast.IntLit:
		return c.compileIntLit(n)
	case *ast.FloatLit:
		return c.compileFloatLit(n)
	case *ast.StringLit:
		return c.compileStringLit(n)
	case *ast.SymbolLit:
		return c.compileSymbolLit(n)
	case *ast.CharLit:
		return c.compileCharLit(n)
	case *ast.ArrayLit:
		return c.compileArrayLit(n)
	case *ast.ByteArrayLit:
		return c.compileByteArrayLit(n)
	case *ast.Ident:
		return c.compileIdent(n)
	case *ast.SelfExpr:
		c.emitOp(OpPushSelf, n.Pos.Line)
	case *ast.SuperExpr:
		// Super is handled at message send time
		c.emitOp(OpPushSelf, n.Pos.Line)
	case *ast.ThisContextExpr:
		// thisContext not implemented yet
		c.emitOp(OpPushNil, n.Pos.Line)
	case *ast.Block:
		return c.compileBlock(n)
	case *ast.VarDecl:
		return c.compileVarDecl(n)
	case *ast.Assign:
		return c.compileAssign(n)
	case *ast.Return:
		return c.compileReturn(n)
	case *ast.UnaryMsg:
		return c.compileUnaryMsg(n)
	case *ast.BinaryMsg:
		return c.compileBinaryMsg(n)
	case *ast.KeywordMsg:
		return c.compileKeywordMsg(n)
	case *ast.Cascade:
		return c.compileCascade(n)
	case *ast.ObjectDecl:
		// Object declarations are handled at the top level
		return nil
	case *ast.InterfaceDecl:
		// Interface declarations are recorded for type checking
		return nil
	case *ast.ImportDecl:
		// Import declarations handled in module loader
		return nil
	case *ast.AnonObjectLit:
		return c.compileAnonObject(n)
	default:
		return fmt.Errorf("compiler: unhandled node type %T", node)
	}
	return nil
}

func (c *Compiler) compileProgram(prog *ast.Program) error {
	for _, stmt := range prog.Statements {
		if err := c.compileNode(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (c *Compiler) compileIntLit(n *ast.IntLit) error {
	// Use OpPushInt for small integers, OpPushConst for large ones
	if n.Value >= math.MinInt32 && n.Value <= math.MaxInt32 {
		c.emitOp(OpPushInt, n.Pos.Line)
		c.chunk.WriteInt32(int32(n.Value), n.Pos.Line)
	} else {
		idx := c.chunk.AddConstant(object.IntObject(n.Value))
		if idx < 0 {
			return fmt.Errorf("constant pool overflow")
		}
		c.emitOp(OpPushConst, n.Pos.Line)
		c.chunk.WriteUint16(uint16(idx), n.Pos.Line)
	}
	return nil
}

func (c *Compiler) compileFloatLit(n *ast.FloatLit) error {
	idx := c.chunk.AddConstant(object.FloatObject(n.Value))
	if idx < 0 {
		return fmt.Errorf("constant pool overflow")
	}
	c.emitOp(OpPushConst, n.Pos.Line)
	c.chunk.WriteUint16(uint16(idx), n.Pos.Line)
	return nil
}

func (c *Compiler) compileStringLit(n *ast.StringLit) error {
	idx := c.chunk.AddConstant(object.StringObject(n.Value))
	if idx < 0 {
		return fmt.Errorf("constant pool overflow")
	}
	c.emitOp(OpPushConst, n.Pos.Line)
	c.chunk.WriteUint16(uint16(idx), n.Pos.Line)
	return nil
}

func (c *Compiler) compileSymbolLit(n *ast.SymbolLit) error {
	idx := c.chunk.AddConstant(object.SymbolObject(n.Value))
	if idx < 0 {
		return fmt.Errorf("constant pool overflow")
	}
	c.emitOp(OpPushConst, n.Pos.Line)
	c.chunk.WriteUint16(uint16(idx), n.Pos.Line)
	return nil
}

func (c *Compiler) compileCharLit(n *ast.CharLit) error {
	idx := c.chunk.AddConstant(object.CharObject(n.Value))
	if idx < 0 {
		return fmt.Errorf("constant pool overflow")
	}
	c.emitOp(OpPushConst, n.Pos.Line)
	c.chunk.WriteUint16(uint16(idx), n.Pos.Line)
	return nil
}

func (c *Compiler) compileArrayLit(n *ast.ArrayLit) error {
	// Compile each element, then create array
	for _, elem := range n.Elements {
		if err := c.compileNode(elem); err != nil {
			return err
		}
	}
	c.emitOp(OpMakeArray, n.Pos.Line)
	c.chunk.WriteUint16(uint16(len(n.Elements)), n.Pos.Line)
	return nil
}

func (c *Compiler) compileByteArrayLit(n *ast.ByteArrayLit) error {
	idx := c.chunk.AddConstant(object.ByteArrayObject(n.Bytes))
	if idx < 0 {
		return fmt.Errorf("constant pool overflow")
	}
	c.emitOp(OpPushConst, n.Pos.Line)
	c.chunk.WriteUint16(uint16(idx), n.Pos.Line)
	return nil
}

func (c *Compiler) compileIdent(n *ast.Ident) error {
	// Check for special names
	switch n.Name {
	case "nil":
		c.emitOp(OpPushNil, n.Pos.Line)
		return nil
	case "true":
		c.emitOp(OpPushTrue, n.Pos.Line)
		return nil
	case "false":
		c.emitOp(OpPushFalse, n.Pos.Line)
		return nil
	case "self":
		c.emitOp(OpPushSelf, n.Pos.Line)
		return nil
	}

	// Try local
	if slot, ok := c.scope.resolveLocal(n.Name); ok {
		c.emitOp(OpPushLocal, n.Pos.Line)
		c.chunk.Write(byte(slot), n.Pos.Line)
		return nil
	}

	// Try upvalue
	if idx, ok := c.scope.resolveUpvalue(n.Name); ok {
		c.emitOp(OpPushUpvalue, n.Pos.Line)
		c.chunk.Write(byte(idx), n.Pos.Line)
		return nil
	}

	// Try instance variable (if in method context)
	if c.isMethod {
		for i, slotName := range c.selfSlotNames {
			if slotName == n.Name {
				idx := c.chunk.AddConstant(object.StringObject(n.Name))
				if idx < 0 {
					return fmt.Errorf("constant pool overflow")
				}
				c.emitOp(OpPushInst, n.Pos.Line)
				c.chunk.WriteUint16(uint16(i), n.Pos.Line)
				return nil
			}
		}
	}

	// Must be a global
	idx := c.chunk.AddConstant(object.StringObject(n.Name))
	if idx < 0 {
		return fmt.Errorf("constant pool overflow")
	}
	c.emitOp(OpPushGlobal, n.Pos.Line)
	c.chunk.WriteUint16(uint16(idx), n.Pos.Line)
	return nil
}

func (c *Compiler) compileBlock(n *ast.Block) error {
	// Create a new compiler for the block with nested scope
	blockCompiler := &Compiler{
		chunk:         NewChunk(),
		scope:         newScope(c.scope),
		blocks:        c.blocks,
		isMethod:      c.isMethod,
		selfSlotNames: c.selfSlotNames,
	}

	// Declare parameters as locals
	for _, param := range n.Params {
		blockCompiler.scope.declareLocal(param)
	}

	// Declare local variables
	for _, local := range n.Locals {
		blockCompiler.scope.declareLocal(local)
	}

	// Compile block body
	for i, stmt := range n.Body {
		if err := blockCompiler.compileNode(stmt); err != nil {
			return err
		}
		// Pop intermediate results except the last
		if i < len(n.Body)-1 {
			blockCompiler.emitOp(OpPop, lineOf(stmt))
		}
	}

	// If empty body, push nil
	if len(n.Body) == 0 {
		blockCompiler.emitOp(OpPushNil, n.Pos.Line)
	}

	blockCompiler.emitOp(OpReturn, n.Pos.Line)

	// Create compiled block
	compiledBlock := &CompiledBlock{
		Arity:      len(n.Params),
		LocalCount: blockCompiler.scope.localCount(),
		Upvalues:   blockCompiler.scope.upvalues,
		Chunk:      blockCompiler.chunk,
		Name:       fmt.Sprintf("block@L%d", n.Pos.Line),
	}

	// Add to blocks list and emit closure instruction
	blockIdx := len(c.blocks)
	c.blocks = append(c.blocks, compiledBlock)

	// Store block in constant pool
	blockObj := &object.Object{
		Kind: object.KindBlock,
	}
	// We'll attach the compiled block data later
	idx := c.chunk.AddConstant(blockObj)
	if idx < 0 {
		return fmt.Errorf("constant pool overflow")
	}

	c.emitOp(OpClosure, n.Pos.Line)
	c.chunk.WriteUint16(uint16(blockIdx), n.Pos.Line)

	return nil
}

func (c *Compiler) compileVarDecl(n *ast.VarDecl) error {
	for _, name := range n.Names {
		c.scope.declareLocal(name)
	}
	return nil
}

func (c *Compiler) compileAssign(n *ast.Assign) error {
	// Compile the value
	if err := c.compileNode(n.Value); err != nil {
		return err
	}

	// Duplicate value (assignment returns the value)
	c.emitOp(OpDup, n.Pos.Line)

	// Store to appropriate location
	if slot, ok := c.scope.resolveLocal(n.Name); ok {
		c.emitOp(OpStoreLocal, n.Pos.Line)
		c.chunk.Write(byte(slot), n.Pos.Line)
		return nil
	}

	if idx, ok := c.scope.resolveUpvalue(n.Name); ok {
		c.emitOp(OpStoreUpvalue, n.Pos.Line)
		c.chunk.Write(byte(idx), n.Pos.Line)
		return nil
	}

	// Check instance variable
	if c.isMethod {
		for i, slotName := range c.selfSlotNames {
			if slotName == n.Name {
				c.emitOp(OpStoreInst, n.Pos.Line)
				c.chunk.WriteUint16(uint16(i), n.Pos.Line)
				return nil
			}
		}
	}

	// Global
	idx := c.chunk.AddConstant(object.StringObject(n.Name))
	if idx < 0 {
		return fmt.Errorf("constant pool overflow")
	}
	c.emitOp(OpStoreGlobal, n.Pos.Line)
	c.chunk.WriteUint16(uint16(idx), n.Pos.Line)
	return nil
}

func (c *Compiler) compileReturn(n *ast.Return) error {
	if err := c.compileNode(n.Value); err != nil {
		return err
	}

	// In a block, ^ is a non-local return
	if c.scope.depth > 0 {
		c.emitOp(OpBlockReturn, n.Pos.Line)
	} else {
		c.emitOp(OpReturn, n.Pos.Line)
	}
	return nil
}

func (c *Compiler) compileUnaryMsg(n *ast.UnaryMsg) error {
	// Check for super send
	if _, isSuper := n.Receiver.(*ast.SuperExpr); isSuper {
		c.emitOp(OpPushSelf, n.Pos.Line)
		idx := c.chunk.AddConstant(object.StringObject(n.Selector))
		if idx < 0 {
			return fmt.Errorf("constant pool overflow")
		}
		c.emitOp(OpSuperSend, n.Pos.Line)
		c.chunk.WriteUint16(uint16(idx), n.Pos.Line)
		c.chunk.Write(0, n.Pos.Line) // argc = 0
		return nil
	}

	// Compile receiver
	if err := c.compileNode(n.Receiver); err != nil {
		return err
	}

	// Emit send
	idx := c.chunk.AddConstant(object.StringObject(n.Selector))
	if idx < 0 {
		return fmt.Errorf("constant pool overflow")
	}
	c.emitOp(OpSend, n.Pos.Line)
	c.chunk.WriteUint16(uint16(idx), n.Pos.Line)
	c.chunk.Write(0, n.Pos.Line) // argc = 0
	return nil
}

func (c *Compiler) compileBinaryMsg(n *ast.BinaryMsg) error {
	// Check for super send
	if _, isSuper := n.Receiver.(*ast.SuperExpr); isSuper {
		c.emitOp(OpPushSelf, n.Pos.Line)
		if err := c.compileNode(n.Arg); err != nil {
			return err
		}
		idx := c.chunk.AddConstant(object.StringObject(n.Op))
		if idx < 0 {
			return fmt.Errorf("constant pool overflow")
		}
		c.emitOp(OpSuperSend, n.Pos.Line)
		c.chunk.WriteUint16(uint16(idx), n.Pos.Line)
		c.chunk.Write(1, n.Pos.Line) // argc = 1
		return nil
	}

	// Compile receiver
	if err := c.compileNode(n.Receiver); err != nil {
		return err
	}

	// Compile argument
	if err := c.compileNode(n.Arg); err != nil {
		return err
	}

	// Emit send
	idx := c.chunk.AddConstant(object.StringObject(n.Op))
	if idx < 0 {
		return fmt.Errorf("constant pool overflow")
	}
	c.emitOp(OpSend, n.Pos.Line)
	c.chunk.WriteUint16(uint16(idx), n.Pos.Line)
	c.chunk.Write(1, n.Pos.Line) // argc = 1
	return nil
}

func (c *Compiler) compileKeywordMsg(n *ast.KeywordMsg) error {
	// Build full selector
	selector := ""
	for _, kw := range n.Keywords {
		selector += kw
	}

	// Check for super send
	if _, isSuper := n.Receiver.(*ast.SuperExpr); isSuper {
		c.emitOp(OpPushSelf, n.Pos.Line)
		for _, arg := range n.Args {
			if err := c.compileNode(arg); err != nil {
				return err
			}
		}
		idx := c.chunk.AddConstant(object.StringObject(selector))
		if idx < 0 {
			return fmt.Errorf("constant pool overflow")
		}
		c.emitOp(OpSuperSend, n.Pos.Line)
		c.chunk.WriteUint16(uint16(idx), n.Pos.Line)
		c.chunk.Write(byte(len(n.Args)), n.Pos.Line)
		return nil
	}

	// Compile receiver
	if err := c.compileNode(n.Receiver); err != nil {
		return err
	}

	// Compile arguments
	for _, arg := range n.Args {
		if err := c.compileNode(arg); err != nil {
			return err
		}
	}

	// Emit send
	idx := c.chunk.AddConstant(object.StringObject(selector))
	if idx < 0 {
		return fmt.Errorf("constant pool overflow")
	}
	c.emitOp(OpSend, n.Pos.Line)
	c.chunk.WriteUint16(uint16(idx), n.Pos.Line)
	c.chunk.Write(byte(len(n.Args)), n.Pos.Line)
	return nil
}

func (c *Compiler) compileCascade(n *ast.Cascade) error {
	// Compile receiver once
	if err := c.compileNode(n.Receiver); err != nil {
		return err
	}

	// For each message except the last, dup receiver, send, pop result
	for i, msg := range n.Messages {
		if i < len(n.Messages)-1 {
			c.emitOp(OpDup, lineOf(msg))
		}

		switch m := msg.(type) {
		case *ast.UnaryMsg:
			idx := c.chunk.AddConstant(object.StringObject(m.Selector))
			if idx < 0 {
				return fmt.Errorf("constant pool overflow")
			}
			c.emitOp(OpSend, m.Pos.Line)
			c.chunk.WriteUint16(uint16(idx), m.Pos.Line)
			c.chunk.Write(0, m.Pos.Line)

		case *ast.BinaryMsg:
			if err := c.compileNode(m.Arg); err != nil {
				return err
			}
			idx := c.chunk.AddConstant(object.StringObject(m.Op))
			if idx < 0 {
				return fmt.Errorf("constant pool overflow")
			}
			c.emitOp(OpSend, m.Pos.Line)
			c.chunk.WriteUint16(uint16(idx), m.Pos.Line)
			c.chunk.Write(1, m.Pos.Line)

		case *ast.KeywordMsg:
			for _, arg := range m.Args {
				if err := c.compileNode(arg); err != nil {
					return err
				}
			}
			selector := ""
			for _, kw := range m.Keywords {
				selector += kw
			}
			idx := c.chunk.AddConstant(object.StringObject(selector))
			if idx < 0 {
				return fmt.Errorf("constant pool overflow")
			}
			c.emitOp(OpSend, m.Pos.Line)
			c.chunk.WriteUint16(uint16(idx), m.Pos.Line)
			c.chunk.Write(byte(len(m.Args)), m.Pos.Line)
		}

		// Pop result for all but last message
		if i < len(n.Messages)-1 {
			c.emitOp(OpPop, lineOf(msg))
		}
	}

	return nil
}

func (c *Compiler) compileAnonObject(n *ast.AnonObjectLit) error {
	// For now, compile anonymous objects as a series of slot assignments
	// This is a simplified implementation

	// Push nil as placeholder for the object
	c.emitOp(OpPushNil, n.Pos.Line)

	// TODO: Implement proper anonymous object compilation
	// This would require creating an object template at compile time

	return nil
}

// CompileMethod compiles a method definition.
func (c *Compiler) CompileMethod(method *ast.MethodDef, slotNames []string) (*CompiledBlock, error) {
	methodCompiler := &Compiler{
		chunk:         NewChunk(),
		scope:         newScope(nil),
		blocks:        make([]*CompiledBlock, 0),
		isMethod:      true,
		selfSlotNames: slotNames,
	}

	// 'self' is implicitly available but not as a local

	// Declare parameters as locals
	for _, param := range method.Params {
		methodCompiler.scope.declareLocal(param)
	}

	// Declare local variables
	for _, local := range method.Locals {
		methodCompiler.scope.declareLocal(local)
	}

	// Compile method body
	for i, stmt := range method.Body {
		if err := methodCompiler.compileNode(stmt); err != nil {
			return nil, err
		}
		if i < len(method.Body)-1 {
			methodCompiler.emitOp(OpPop, lineOf(stmt))
		}
	}

	// Methods implicitly return self if no explicit return
	if len(method.Body) == 0 {
		methodCompiler.emitOp(OpReturnSelf, method.Pos.Line)
	} else {
		methodCompiler.emitOp(OpReturnSelf, method.Pos.Line)
	}

	return &CompiledBlock{
		Arity:      len(method.Params),
		LocalCount: methodCompiler.scope.localCount(),
		Upvalues:   nil, // Methods don't capture upvalues
		Chunk:      methodCompiler.chunk,
		Name:       method.Selector,
	}, nil
}

// GetBlocks returns all compiled blocks (for closure creation).
func (c *Compiler) GetBlocks() []*CompiledBlock {
	return c.blocks
}

// Helper methods

func (c *Compiler) emitOp(op OpCode, line int) {
	c.chunk.WriteOp(op, line)
}

func lineOf(n ast.Node) int {
	switch node := n.(type) {
	case *ast.NilLit:
		return node.Pos.Line
	case *ast.BoolLit:
		return node.Pos.Line
	case *ast.IntLit:
		return node.Pos.Line
	case *ast.FloatLit:
		return node.Pos.Line
	case *ast.StringLit:
		return node.Pos.Line
	case *ast.SymbolLit:
		return node.Pos.Line
	case *ast.CharLit:
		return node.Pos.Line
	case *ast.ArrayLit:
		return node.Pos.Line
	case *ast.ByteArrayLit:
		return node.Pos.Line
	case *ast.Ident:
		return node.Pos.Line
	case *ast.SelfExpr:
		return node.Pos.Line
	case *ast.SuperExpr:
		return node.Pos.Line
	case *ast.Block:
		return node.Pos.Line
	case *ast.VarDecl:
		return node.Pos.Line
	case *ast.Assign:
		return node.Pos.Line
	case *ast.Return:
		return node.Pos.Line
	case *ast.UnaryMsg:
		return node.Pos.Line
	case *ast.BinaryMsg:
		return node.Pos.Line
	case *ast.KeywordMsg:
		return node.Pos.Line
	case *ast.Cascade:
		return node.Pos.Line
	case *ast.Program:
		return node.Pos.Line
	default:
		return 1
	}
}
