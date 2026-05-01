// Package bytecode defines the picoceci bytecode instruction set,
// the compiler (AST → bytecode), and the bytecode virtual machine.
package bytecode

import "fmt"

// OpCode represents a single bytecode instruction.
type OpCode uint8

const (
	// Stack manipulation
	OpPop OpCode = iota // discard TOS
	OpDup               // duplicate TOS

	// Push constants
	OpPushNil   // push nil
	OpPushTrue  // push true
	OpPushFalse // push false
	OpPushSelf  // push self
	OpPushInt   // push int32 immediate (next 4 bytes)
	OpPushConst // push constant from pool (next 2 bytes = index)

	// Variables
	OpPushLocal    // push local[arg] (next 1 byte = slot)
	OpStoreLocal   // TOS → local[arg], pop (next 1 byte = slot)
	OpPushUpvalue  // push upvalue[arg] (next 1 byte = index)
	OpStoreUpvalue // TOS → upvalue[arg], pop (next 1 byte = index)
	OpPushInst     // push self.slots[arg] (next 2 bytes = name index)
	OpStoreInst    // TOS → self.slots[arg], pop (next 2 bytes = name index)
	OpPushGlobal   // push global[arg] (next 2 bytes = name index)
	OpStoreGlobal  // TOS → global[arg], pop (next 2 bytes = name index)

	// Message sends
	OpSend      // send message (next: 2 bytes selector idx, 1 byte argc)
	OpSuperSend // super send (next: 2 bytes selector idx, 1 byte argc)

	// Blocks and closures
	OpClosure // create closure from CompiledBlock (next 2 bytes = block index)

	// Control flow
	OpJump        // unconditional jump (next 2 bytes = signed offset)
	OpJumpIfFalse // pop, jump if false (next 2 bytes = signed offset)
	OpJumpIfTrue  // pop, jump if true (next 2 bytes = signed offset)

	// Return
	OpReturn      // return TOS
	OpReturnSelf  // return self
	OpBlockReturn // non-local return from block

	// Array operations (optimization for literal arrays)
	OpMakeArray // create array from N stack items (next 2 bytes = count)
)

// opCodeNames maps opcodes to their string names.
var opCodeNames = [...]string{
	OpPop:          "POP",
	OpDup:          "DUP",
	OpPushNil:      "PUSH_NIL",
	OpPushTrue:     "PUSH_TRUE",
	OpPushFalse:    "PUSH_FALSE",
	OpPushSelf:     "PUSH_SELF",
	OpPushInt:      "PUSH_INT",
	OpPushConst:    "PUSH_CONST",
	OpPushLocal:    "PUSH_LOCAL",
	OpStoreLocal:   "STORE_LOCAL",
	OpPushUpvalue:  "PUSH_UPVALUE",
	OpStoreUpvalue: "STORE_UPVALUE",
	OpPushInst:     "PUSH_INST",
	OpStoreInst:    "STORE_INST",
	OpPushGlobal:   "PUSH_GLOBAL",
	OpStoreGlobal:  "STORE_GLOBAL",
	OpSend:         "SEND",
	OpSuperSend:    "SUPER_SEND",
	OpClosure:      "CLOSURE",
	OpJump:         "JUMP",
	OpJumpIfFalse:  "JUMP_IF_FALSE",
	OpJumpIfTrue:   "JUMP_IF_TRUE",
	OpReturn:       "RETURN",
	OpReturnSelf:   "RETURN_SELF",
	OpBlockReturn:  "BLOCK_RETURN",
	OpMakeArray:    "MAKE_ARRAY",
}

// String returns the human-readable name of an opcode.
func (op OpCode) String() string {
	if int(op) < len(opCodeNames) {
		return opCodeNames[op]
	}
	return fmt.Sprintf("UNKNOWN(%d)", op)
}

// OperandWidths returns the byte widths of operands for this opcode.
// For example, OpPushInt returns [4] (one 4-byte operand).
// An empty slice means no operands.
func (op OpCode) OperandWidths() []int {
	switch op {
	case OpPop, OpDup, OpPushNil, OpPushTrue, OpPushFalse, OpPushSelf,
		OpReturn, OpReturnSelf, OpBlockReturn:
		return nil // no operands

	case OpPushLocal, OpStoreLocal, OpPushUpvalue, OpStoreUpvalue:
		return []int{1} // 1-byte slot index

	case OpPushConst, OpPushInst, OpStoreInst, OpPushGlobal, OpStoreGlobal,
		OpClosure, OpJump, OpJumpIfFalse, OpJumpIfTrue, OpMakeArray:
		return []int{2} // 2-byte index or offset

	case OpPushInt:
		return []int{4} // 4-byte immediate integer

	case OpSend, OpSuperSend:
		return []int{2, 1} // 2-byte selector index, 1-byte argc

	default:
		return nil
	}
}

// InstructionLength returns the total byte length of an instruction
// including the opcode byte and all operands.
func (op OpCode) InstructionLength() int {
	length := 1 // opcode itself
	for _, w := range op.OperandWidths() {
		length += w
	}
	return length
}
