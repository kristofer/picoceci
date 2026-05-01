package bytecode

import (
	"fmt"
	"strings"

	"github.com/kristofer/picoceci/pkg/object"
)

// Chunk holds compiled bytecode and associated constant pool.
type Chunk struct {
	Code      []byte           // bytecode instructions
	Constants []*object.Object // constant pool (strings, floats, symbols, blocks)
	Lines     []int            // source line for each byte (for error messages)
}

// NewChunk creates a new empty chunk.
func NewChunk() *Chunk {
	return &Chunk{
		Code:      make([]byte, 0, 256),
		Constants: make([]*object.Object, 0, 16),
		Lines:     make([]int, 0, 256),
	}
}

// Write appends a byte to the chunk.
func (c *Chunk) Write(b byte, line int) {
	c.Code = append(c.Code, b)
	c.Lines = append(c.Lines, line)
}

// WriteOp appends an opcode to the chunk.
func (c *Chunk) WriteOp(op OpCode, line int) {
	c.Write(byte(op), line)
}

// WriteUint16 appends a 16-bit value in big-endian order.
func (c *Chunk) WriteUint16(v uint16, line int) {
	c.Write(byte(v>>8), line)
	c.Write(byte(v), line)
}

// WriteInt32 appends a 32-bit value in big-endian order.
func (c *Chunk) WriteInt32(v int32, line int) {
	c.Write(byte(v>>24), line)
	c.Write(byte(v>>16), line)
	c.Write(byte(v>>8), line)
	c.Write(byte(v), line)
}

// AddConstant adds a constant to the pool and returns its index.
// Returns -1 if the constant pool is full (max 65535 constants).
func (c *Chunk) AddConstant(val *object.Object) int {
	if len(c.Constants) >= 65535 {
		return -1
	}
	c.Constants = append(c.Constants, val)
	return len(c.Constants) - 1
}

// ReadUint16 reads a 16-bit value at the given offset.
func (c *Chunk) ReadUint16(offset int) uint16 {
	return uint16(c.Code[offset])<<8 | uint16(c.Code[offset+1])
}

// ReadInt16 reads a signed 16-bit value at the given offset.
func (c *Chunk) ReadInt16(offset int) int16 {
	return int16(c.ReadUint16(offset))
}

// ReadInt32 reads a 32-bit value at the given offset.
func (c *Chunk) ReadInt32(offset int) int32 {
	return int32(c.Code[offset])<<24 |
		int32(c.Code[offset+1])<<16 |
		int32(c.Code[offset+2])<<8 |
		int32(c.Code[offset+3])
}

// Disassemble returns a human-readable bytecode listing.
func (c *Chunk) Disassemble(name string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("== %s ==\n", name))

	for offset := 0; offset < len(c.Code); {
		offset = c.disassembleInstruction(&sb, offset)
	}

	return sb.String()
}

// disassembleInstruction disassembles one instruction and returns the next offset.
func (c *Chunk) disassembleInstruction(sb *strings.Builder, offset int) int {
	sb.WriteString(fmt.Sprintf("%04d ", offset))

	// Print line number (or | if same as previous)
	if offset > 0 && c.Lines[offset] == c.Lines[offset-1] {
		sb.WriteString("   | ")
	} else {
		sb.WriteString(fmt.Sprintf("%4d ", c.Lines[offset]))
	}

	op := OpCode(c.Code[offset])
	sb.WriteString(op.String())

	widths := op.OperandWidths()
	operandOffset := offset + 1

	for i, width := range widths {
		var val int
		switch width {
		case 1:
			val = int(c.Code[operandOffset])
		case 2:
			val = int(c.ReadUint16(operandOffset))
		case 4:
			val = int(c.ReadInt32(operandOffset))
		}

		sb.WriteString(fmt.Sprintf(" %d", val))

		// For constant pool references, show the constant value
		if i == 0 && (op == OpPushConst || op == OpPushGlobal || op == OpStoreGlobal ||
			op == OpPushInst || op == OpStoreInst || op == OpSend || op == OpSuperSend ||
			op == OpClosure) {
			if val >= 0 && val < len(c.Constants) {
				sb.WriteString(fmt.Sprintf(" (%s)", c.Constants[val].PrintString()))
			}
		}

		operandOffset += width
	}

	sb.WriteString("\n")
	return operandOffset
}

// PatchJump patches a jump instruction at the given offset with the current position.
// Used for forward jumps where the target isn't known at emit time.
func (c *Chunk) PatchJump(offset int) {
	// Calculate the jump distance (from after the jump instruction to current position)
	jump := len(c.Code) - offset - 2 // -2 for the 2-byte operand
	if jump > 32767 || jump < -32768 {
		// Jump too large - this is a compilation error
		// In a real implementation, we'd handle this more gracefully
		panic("jump offset too large")
	}
	c.Code[offset] = byte(jump >> 8)
	c.Code[offset+1] = byte(jump)
}

// CurrentOffset returns the current position in the bytecode.
func (c *Chunk) CurrentOffset() int {
	return len(c.Code)
}

// Upvalue describes a captured variable from an enclosing scope.
type Upvalue struct {
	Index   uint8 // slot index in parent's locals or upvalues
	IsLocal bool  // true if captured from immediate parent's locals
}

// CompiledBlock represents a compiled block/closure template.
type CompiledBlock struct {
	Arity      int       // number of parameters
	LocalCount int       // number of local variables (including params)
	Upvalues   []Upvalue // captured variable descriptors
	Chunk      *Chunk    // the bytecode
	Name       string    // for debugging (e.g., "block in Counter>>increment")
}

// NewCompiledBlock creates a new compiled block.
func NewCompiledBlock(name string, arity int) *CompiledBlock {
	return &CompiledBlock{
		Arity:    arity,
		Upvalues: make([]Upvalue, 0),
		Chunk:    NewChunk(),
		Name:     name,
	}
}

// AddUpvalue adds an upvalue descriptor and returns its index.
func (cb *CompiledBlock) AddUpvalue(index uint8, isLocal bool) int {
	// Check if we already have this upvalue
	for i, uv := range cb.Upvalues {
		if uv.Index == index && uv.IsLocal == isLocal {
			return i
		}
	}
	cb.Upvalues = append(cb.Upvalues, Upvalue{Index: index, IsLocal: isLocal})
	return len(cb.Upvalues) - 1
}

// CompiledMethod represents a compiled method.
type CompiledMethod struct {
	Selector string         // e.g., "increment" or "at:put:"
	Block    *CompiledBlock // the method body
}

// CompiledObject represents a compiled object template.
type CompiledObject struct {
	Name      string                     // object name
	SlotNames []string                   // instance variable names
	Methods   map[string]*CompiledMethod // compiled methods
	Composes  []string                   // composed object names
}

// NewCompiledObject creates a new compiled object template.
func NewCompiledObject(name string) *CompiledObject {
	return &CompiledObject{
		Name:      name,
		SlotNames: make([]string, 0),
		Methods:   make(map[string]*CompiledMethod),
		Composes:  make([]string, 0),
	}
}
