package bytecode

import (
	"strings"
	"testing"

	"github.com/kristofer/picoceci/pkg/object"
)

func TestChunkWrite(t *testing.T) {
	c := NewChunk()
	c.WriteOp(OpPushNil, 1)
	c.WriteOp(OpReturn, 1)

	if len(c.Code) != 2 {
		t.Errorf("expected 2 bytes, got %d", len(c.Code))
	}
	if OpCode(c.Code[0]) != OpPushNil {
		t.Errorf("expected OpPushNil, got %v", OpCode(c.Code[0]))
	}
	if OpCode(c.Code[1]) != OpReturn {
		t.Errorf("expected OpReturn, got %v", OpCode(c.Code[1]))
	}
}

func TestChunkWriteUint16(t *testing.T) {
	c := NewChunk()
	c.WriteUint16(0x1234, 1)

	if len(c.Code) != 2 {
		t.Errorf("expected 2 bytes, got %d", len(c.Code))
	}
	if c.Code[0] != 0x12 || c.Code[1] != 0x34 {
		t.Errorf("expected [0x12, 0x34], got [0x%02x, 0x%02x]", c.Code[0], c.Code[1])
	}

	val := c.ReadUint16(0)
	if val != 0x1234 {
		t.Errorf("ReadUint16 expected 0x1234, got 0x%04x", val)
	}
}

func TestChunkWriteInt32(t *testing.T) {
	c := NewChunk()
	c.WriteInt32(0x12345678, 1)

	if len(c.Code) != 4 {
		t.Errorf("expected 4 bytes, got %d", len(c.Code))
	}

	val := c.ReadInt32(0)
	if val != 0x12345678 {
		t.Errorf("ReadInt32 expected 0x12345678, got 0x%08x", val)
	}
}

func TestChunkWriteInt32Negative(t *testing.T) {
	c := NewChunk()
	c.WriteInt32(-42, 1)

	val := c.ReadInt32(0)
	if val != -42 {
		t.Errorf("ReadInt32 expected -42, got %d", val)
	}
}

func TestChunkAddConstant(t *testing.T) {
	c := NewChunk()

	idx1 := c.AddConstant(object.IntObject(42))
	idx2 := c.AddConstant(object.StringObject("hello"))

	if idx1 != 0 {
		t.Errorf("first constant index expected 0, got %d", idx1)
	}
	if idx2 != 1 {
		t.Errorf("second constant index expected 1, got %d", idx2)
	}

	if c.Constants[0].IVal != 42 {
		t.Errorf("first constant expected 42, got %d", c.Constants[0].IVal)
	}
	if c.Constants[1].SVal != "hello" {
		t.Errorf("second constant expected 'hello', got %q", c.Constants[1].SVal)
	}
}

func TestChunkDisassemble(t *testing.T) {
	c := NewChunk()

	// Add some constants
	c.AddConstant(object.IntObject(42))
	c.AddConstant(object.StringObject("hello"))

	// Write some instructions
	c.WriteOp(OpPushConst, 1)
	c.WriteUint16(0, 1) // constant index 0

	c.WriteOp(OpPushConst, 2)
	c.WriteUint16(1, 2) // constant index 1

	c.WriteOp(OpReturn, 3)

	dis := c.Disassemble("test")

	// Check that disassembly contains expected parts
	if !strings.Contains(dis, "== test ==") {
		t.Error("disassembly missing header")
	}
	if !strings.Contains(dis, "PUSH_CONST") {
		t.Error("disassembly missing PUSH_CONST")
	}
	if !strings.Contains(dis, "RETURN") {
		t.Error("disassembly missing RETURN")
	}
	if !strings.Contains(dis, "42") {
		t.Error("disassembly missing constant value 42")
	}
	if !strings.Contains(dis, "'hello'") {
		t.Error("disassembly missing constant value 'hello'")
	}
}

func TestChunkPatchJump(t *testing.T) {
	c := NewChunk()

	// Emit a jump with placeholder
	c.WriteOp(OpJump, 1)
	jumpOffset := c.CurrentOffset()
	c.WriteUint16(0, 1) // placeholder

	// Emit some more code
	c.WriteOp(OpPushNil, 2)
	c.WriteOp(OpPop, 2)

	// Patch the jump
	c.PatchJump(jumpOffset)

	// The jump should now point past the PushNil and Pop
	jumpDist := c.ReadInt16(jumpOffset)
	expected := int16(2) // 2 bytes (OpPushNil + OpPop)
	if jumpDist != expected {
		t.Errorf("expected jump distance %d, got %d", expected, jumpDist)
	}
}

func TestCompiledBlockAddUpvalue(t *testing.T) {
	cb := NewCompiledBlock("test", 0)

	// Add first upvalue
	idx1 := cb.AddUpvalue(0, true)
	if idx1 != 0 {
		t.Errorf("first upvalue index expected 0, got %d", idx1)
	}

	// Add second upvalue
	idx2 := cb.AddUpvalue(1, false)
	if idx2 != 1 {
		t.Errorf("second upvalue index expected 1, got %d", idx2)
	}

	// Adding same upvalue should return existing index
	idx3 := cb.AddUpvalue(0, true)
	if idx3 != 0 {
		t.Errorf("duplicate upvalue should return 0, got %d", idx3)
	}

	if len(cb.Upvalues) != 2 {
		t.Errorf("expected 2 upvalues, got %d", len(cb.Upvalues))
	}
}

func TestCompiledObject(t *testing.T) {
	co := NewCompiledObject("Counter")
	co.SlotNames = []string{"count"}
	co.Composes = []string{"Printable"}

	method := &CompiledMethod{
		Selector: "increment",
		Block:    NewCompiledBlock("Counter>>increment", 0),
	}
	co.Methods["increment"] = method

	if co.Name != "Counter" {
		t.Errorf("expected name 'Counter', got %q", co.Name)
	}
	if len(co.SlotNames) != 1 || co.SlotNames[0] != "count" {
		t.Errorf("unexpected slot names: %v", co.SlotNames)
	}
	if len(co.Composes) != 1 || co.Composes[0] != "Printable" {
		t.Errorf("unexpected composes: %v", co.Composes)
	}
	if _, ok := co.Methods["increment"]; !ok {
		t.Error("expected 'increment' method")
	}
}
