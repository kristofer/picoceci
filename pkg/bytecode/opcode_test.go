package bytecode

import "testing"

func TestOpCodeString(t *testing.T) {
	tests := []struct {
		op   OpCode
		want string
	}{
		{OpPop, "POP"},
		{OpDup, "DUP"},
		{OpPushNil, "PUSH_NIL"},
		{OpPushTrue, "PUSH_TRUE"},
		{OpPushFalse, "PUSH_FALSE"},
		{OpPushSelf, "PUSH_SELF"},
		{OpPushInt, "PUSH_INT"},
		{OpPushConst, "PUSH_CONST"},
		{OpPushLocal, "PUSH_LOCAL"},
		{OpStoreLocal, "STORE_LOCAL"},
		{OpPushUpvalue, "PUSH_UPVALUE"},
		{OpStoreUpvalue, "STORE_UPVALUE"},
		{OpPushInst, "PUSH_INST"},
		{OpStoreInst, "STORE_INST"},
		{OpPushGlobal, "PUSH_GLOBAL"},
		{OpStoreGlobal, "STORE_GLOBAL"},
		{OpSend, "SEND"},
		{OpSuperSend, "SUPER_SEND"},
		{OpClosure, "CLOSURE"},
		{OpJump, "JUMP"},
		{OpJumpIfFalse, "JUMP_IF_FALSE"},
		{OpJumpIfTrue, "JUMP_IF_TRUE"},
		{OpReturn, "RETURN"},
		{OpReturnSelf, "RETURN_SELF"},
		{OpBlockReturn, "BLOCK_RETURN"},
		{OpMakeArray, "MAKE_ARRAY"},
	}

	for _, tt := range tests {
		if got := tt.op.String(); got != tt.want {
			t.Errorf("OpCode(%d).String() = %q, want %q", tt.op, got, tt.want)
		}
	}
}

func TestOpCodeUnknown(t *testing.T) {
	unknown := OpCode(255)
	got := unknown.String()
	if got != "UNKNOWN(255)" {
		t.Errorf("unknown opcode String() = %q, want %q", got, "UNKNOWN(255)")
	}
}

func TestOperandWidths(t *testing.T) {
	tests := []struct {
		op   OpCode
		want []int
	}{
		// No operands
		{OpPop, nil},
		{OpDup, nil},
		{OpPushNil, nil},
		{OpPushTrue, nil},
		{OpPushFalse, nil},
		{OpPushSelf, nil},
		{OpReturn, nil},
		{OpReturnSelf, nil},
		{OpBlockReturn, nil},

		// 1-byte operand
		{OpPushLocal, []int{1}},
		{OpStoreLocal, []int{1}},
		{OpPushUpvalue, []int{1}},
		{OpStoreUpvalue, []int{1}},

		// 2-byte operand
		{OpPushConst, []int{2}},
		{OpPushInst, []int{2}},
		{OpStoreInst, []int{2}},
		{OpPushGlobal, []int{2}},
		{OpStoreGlobal, []int{2}},
		{OpClosure, []int{2}},
		{OpJump, []int{2}},
		{OpJumpIfFalse, []int{2}},
		{OpJumpIfTrue, []int{2}},
		{OpMakeArray, []int{2}},

		// 4-byte operand
		{OpPushInt, []int{4}},

		// Multiple operands
		{OpSend, []int{2, 1}},
		{OpSuperSend, []int{2, 1}},
	}

	for _, tt := range tests {
		got := tt.op.OperandWidths()
		if !sliceEqual(got, tt.want) {
			t.Errorf("%s.OperandWidths() = %v, want %v", tt.op, got, tt.want)
		}
	}
}

func TestInstructionLength(t *testing.T) {
	tests := []struct {
		op   OpCode
		want int
	}{
		{OpPop, 1},
		{OpPushNil, 1},
		{OpPushLocal, 2},  // 1 + 1
		{OpPushConst, 3},  // 1 + 2
		{OpPushInt, 5},    // 1 + 4
		{OpSend, 4},       // 1 + 2 + 1
		{OpSuperSend, 4},  // 1 + 2 + 1
	}

	for _, tt := range tests {
		if got := tt.op.InstructionLength(); got != tt.want {
			t.Errorf("%s.InstructionLength() = %d, want %d", tt.op, got, tt.want)
		}
	}
}

func sliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
