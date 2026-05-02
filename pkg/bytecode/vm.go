package bytecode

import (
	"fmt"

	"github.com/kristofer/picoceci/pkg/ast"
	"github.com/kristofer/picoceci/pkg/eval"
	"github.com/kristofer/picoceci/pkg/object"
)

const (
	maxStackSize  = 1024
	maxFrameCount = 256
)

// VM executes compiled bytecode.
type VM struct {
	globals    map[string]*object.Object // global namespace
	stack      [maxStackSize]*object.Object
	sp         int // stack pointer (points to next free slot)
	frames     [maxFrameCount]CallFrame
	frameCount int
	blocks     []*CompiledBlock // compiled block templates
}

// CallFrame represents a single activation record.
type CallFrame struct {
	closure   *Closure // the closure being executed
	ip        int      // instruction pointer within chunk
	bp        int      // base pointer (stack frame start)
	selfObj   *object.Object // 'self' for method calls
}

// Closure is a runtime closure (CompiledBlock + captured upvalues).
type Closure struct {
	Block    *CompiledBlock
	Upvalues []*UpvalueRef
}

// UpvalueRef points to a captured variable.
// When open (Closed=false), it references a stack slot via StackSlot and VM.
// When closed (Closed=true), it stores the value directly in Value.
type UpvalueRef struct {
	Value     *object.Object // the captured value (when closed)
	Closed    bool           // true once the enclosing scope has exited
	StackSlot int            // stack slot index (when open)
	VM        *VM            // reference to VM for stack access (when open)
}

// NewVM creates a new VM with default globals.
func NewVM() *VM {
	vm := &VM{
		globals: eval.InitialGlobals(),
		sp:      0,
		blocks:  make([]*CompiledBlock, 0),
	}
	return vm
}

// SetBlocks sets the compiled block templates for closure creation.
func (vm *VM) SetBlocks(blocks []*CompiledBlock) {
	vm.blocks = blocks
}

// AddGlobals merges additional globals into the VM's global namespace.
// This is used for adding globals from imported modules.
func (vm *VM) AddGlobals(globals map[string]*object.Object) {
	for name, obj := range globals {
		vm.globals[name] = obj
	}
}

// Run executes the given chunk and returns the result.
func (vm *VM) Run(chunk *Chunk) (*object.Object, error) {
	// Create a main closure
	mainBlock := &CompiledBlock{
		Arity:      0,
		LocalCount: 0,
		Chunk:      chunk,
		Name:       "<main>",
	}
	mainClosure := &Closure{Block: mainBlock}

	// Push initial frame
	vm.frames[0] = CallFrame{
		closure: mainClosure,
		ip:      0,
		bp:      0,
	}
	vm.frameCount = 1

	return vm.run()
}

// CallBlock implements eval.BlockCaller interface.
func (vm *VM) CallBlock(blk *object.Object, args []*object.Object) (*object.Object, error) {
	if blk.Kind != object.KindBlock {
		return nil, fmt.Errorf("not a block")
	}

	// Get the closure from the block object
	closure, ok := blk.Env.(*Closure)
	if !ok {
		// Fall back to tree-walking interpreter's block calling if no compiled closure
		// This shouldn't happen in pure bytecode execution
		return nil, fmt.Errorf("block has no compiled closure")
	}

	// Push arguments onto stack
	for _, arg := range args {
		vm.push(arg)
	}

	// Save current state
	savedFrameCount := vm.frameCount

	// Create new frame for block
	frame := &vm.frames[vm.frameCount]
	frame.closure = closure
	frame.ip = 0
	frame.bp = vm.sp - len(args)
	frame.selfObj = nil
	vm.frameCount++

	// Allocate space for locals beyond parameters
	for i := len(args); i < closure.Block.LocalCount; i++ {
		vm.push(object.Nil)
	}

	// Run until this frame returns
	result, err := vm.runFrame(savedFrameCount)

	return result, err
}

// run is the main execution loop.
func (vm *VM) run() (*object.Object, error) {
	for vm.frameCount > 0 {
		result, done, err := vm.step()
		if err != nil {
			return nil, err
		}
		if done {
			return result, nil
		}
	}
	return object.Nil, nil
}

// runFrame runs until the frame count drops to savedFrameCount.
func (vm *VM) runFrame(savedFrameCount int) (*object.Object, error) {
	for vm.frameCount > savedFrameCount {
		result, done, err := vm.step()
		if err != nil {
			return nil, err
		}
		if done {
			return result, nil
		}
	}
	// When the block returns, the result is on the stack
	if vm.sp > 0 {
		return vm.pop(), nil
	}
	return object.Nil, nil
}

// step executes a single instruction.
func (vm *VM) step() (*object.Object, bool, error) {
	frame := &vm.frames[vm.frameCount-1]
	chunk := frame.closure.Block.Chunk

	if frame.ip >= len(chunk.Code) {
		// Implicit return nil at end of chunk
		vm.frameCount--
		return object.Nil, vm.frameCount == 0, nil
	}

	op := OpCode(chunk.Code[frame.ip])
	frame.ip++

	switch op {
	case OpPop:
		vm.pop()

	case OpDup:
		vm.push(vm.peek(0))

	case OpPushNil:
		vm.push(object.Nil)

	case OpPushTrue:
		vm.push(object.True)

	case OpPushFalse:
		vm.push(object.False)

	case OpPushSelf:
		if frame.selfObj != nil {
			vm.push(frame.selfObj)
		} else {
			vm.push(object.Nil)
		}

	case OpPushInt:
		val := chunk.ReadInt32(frame.ip)
		frame.ip += 4
		vm.push(object.IntObject(int64(val)))

	case OpPushConst:
		idx := chunk.ReadUint16(frame.ip)
		frame.ip += 2
		vm.push(chunk.Constants[idx])

	case OpPushLocal:
		slot := int(chunk.Code[frame.ip])
		frame.ip++
		vm.push(vm.stack[frame.bp+slot])

	case OpStoreLocal:
		slot := int(chunk.Code[frame.ip])
		frame.ip++
		vm.stack[frame.bp+slot] = vm.pop()

	case OpPushUpvalue:
		idx := int(chunk.Code[frame.ip])
		frame.ip++
		if idx < len(frame.closure.Upvalues) {
			uv := frame.closure.Upvalues[idx]
			if uv.Closed {
				vm.push(uv.Value)
			} else {
				vm.push(vm.stack[uv.StackSlot])
			}
		} else {
			vm.push(object.Nil)
		}

	case OpStoreUpvalue:
		idx := int(chunk.Code[frame.ip])
		frame.ip++
		val := vm.pop()
		if idx < len(frame.closure.Upvalues) {
			uv := frame.closure.Upvalues[idx]
			if uv.Closed {
				uv.Value = val
			} else {
				vm.stack[uv.StackSlot] = val
			}
		}

	case OpPushInst:
		idx := chunk.ReadUint16(frame.ip)
		frame.ip += 2
		if frame.selfObj != nil && frame.selfObj.Slots != nil {
			name := chunk.Constants[idx].SVal
			if v, ok := frame.selfObj.Slots[name]; ok {
				vm.push(v)
			} else {
				vm.push(object.Nil)
			}
		} else {
			vm.push(object.Nil)
		}

	case OpStoreInst:
		idx := chunk.ReadUint16(frame.ip)
		frame.ip += 2
		val := vm.pop()
		if frame.selfObj != nil && frame.selfObj.Slots != nil {
			name := chunk.Constants[idx].SVal
			frame.selfObj.Slots[name] = val
		}

	case OpPushGlobal:
		idx := chunk.ReadUint16(frame.ip)
		frame.ip += 2
		name := chunk.Constants[idx].SVal
		if v, ok := vm.globals[name]; ok {
			vm.push(v)
		} else {
			vm.push(object.Nil)
		}

	case OpStoreGlobal:
		idx := chunk.ReadUint16(frame.ip)
		frame.ip += 2
		name := chunk.Constants[idx].SVal
		vm.globals[name] = vm.pop()

	case OpSend:
		selIdx := chunk.ReadUint16(frame.ip)
		frame.ip += 2
		argc := int(chunk.Code[frame.ip])
		frame.ip++

		selector := chunk.Constants[selIdx].SVal
		result, err := vm.send(selector, argc)
		if err != nil {
			return nil, false, err
		}
		vm.push(result)

	case OpSuperSend:
		selIdx := chunk.ReadUint16(frame.ip)
		frame.ip += 2
		argc := int(chunk.Code[frame.ip])
		frame.ip++

		selector := chunk.Constants[selIdx].SVal
		result, err := vm.superSend(selector, argc)
		if err != nil {
			return nil, false, err
		}
		vm.push(result)

	case OpClosure:
		blockIdx := chunk.ReadUint16(frame.ip)
		frame.ip += 2

		if int(blockIdx) >= len(vm.blocks) {
			return nil, false, fmt.Errorf("invalid block index %d", blockIdx)
		}

		block := vm.blocks[blockIdx]
		closure := &Closure{
			Block:    block,
			Upvalues: make([]*UpvalueRef, len(block.Upvalues)),
		}

		// Capture upvalues
		for i, uv := range block.Upvalues {
			if uv.IsLocal {
				// Capture from current frame's locals - create open upvalue
				slot := frame.bp + int(uv.Index)
				closure.Upvalues[i] = &UpvalueRef{
					StackSlot: slot,
					VM:        vm,
					Closed:    false, // Open upvalue references stack slot
				}
			} else {
				// Capture from current frame's upvalues
				if int(uv.Index) < len(frame.closure.Upvalues) {
					closure.Upvalues[i] = frame.closure.Upvalues[uv.Index]
				} else {
					closure.Upvalues[i] = &UpvalueRef{Value: object.Nil, Closed: true}
				}
			}
		}

		// Create block object with closure attached
		blkObj := &object.Object{
			Kind:   object.KindBlock,
			Params: make([]string, block.Arity),
			Env:    closure,
		}
		vm.push(blkObj)

	case OpJump:
		offset := chunk.ReadInt16(frame.ip)
		frame.ip += 2
		frame.ip += int(offset)

	case OpJumpIfFalse:
		offset := chunk.ReadInt16(frame.ip)
		frame.ip += 2
		cond := vm.pop()
		if !cond.Truthy() {
			frame.ip += int(offset)
		}

	case OpJumpIfTrue:
		offset := chunk.ReadInt16(frame.ip)
		frame.ip += 2
		cond := vm.pop()
		if cond.Truthy() {
			frame.ip += int(offset)
		}

	case OpReturn:
		result := vm.pop()
		vm.frameCount--
		if vm.frameCount == 0 {
			return result, true, nil
		}
		// Pop locals from stack
		vm.sp = frame.bp
		vm.push(result)

	case OpReturnSelf:
		result := frame.selfObj
		if result == nil {
			result = object.Nil
		}
		vm.frameCount--
		if vm.frameCount == 0 {
			return result, true, nil
		}
		vm.sp = frame.bp
		vm.push(result)

	case OpBlockReturn:
		// Non-local return - for now, treat as regular return
		// Full implementation would unwind to the enclosing method
		result := vm.pop()
		vm.frameCount--
		if vm.frameCount == 0 {
			return result, true, nil
		}
		vm.sp = frame.bp
		vm.push(result)

	case OpMakeArray:
		count := int(chunk.ReadUint16(frame.ip))
		frame.ip += 2
		arr := object.ArrayObject(count)
		for i := count - 1; i >= 0; i-- {
			arr.Items[i] = vm.pop()
		}
		vm.push(arr)

	default:
		return nil, false, fmt.Errorf("unknown opcode: %d", op)
	}

	return nil, false, nil
}

// send performs a message send.
func (vm *VM) send(selector string, argc int) (*object.Object, error) {
	// Get arguments and receiver from stack
	args := make([]*object.Object, argc)
	for i := argc - 1; i >= 0; i-- {
		args[i] = vm.pop()
	}
	recv := vm.pop()

	if recv == nil {
		recv = object.Nil
	}

	// Try object's methods first
	if recv.Methods != nil {
		if m, ok := recv.Methods[selector]; ok {
			return vm.applyMethod(recv, m, args)
		}
	}

	// Try builtin dispatch
	pos := ast.Pos{Line: 1, Col: 1} // We don't have precise position in VM
	result, err, handled := eval.BuiltinDispatch(vm, recv, selector, args, pos)
	if handled {
		return result, err
	}

	return nil, &eval.Error{
		Kind:    "MessageNotUnderstood",
		Message: fmt.Sprintf("%s does not understand #%s", recv.PrintString(), selector),
		Pos:     pos,
	}
}

// superSend performs a super message send.
func (vm *VM) superSend(selector string, argc int) (*object.Object, error) {
	// Get arguments from stack (receiver is self)
	args := make([]*object.Object, argc)
	for i := argc - 1; i >= 0; i-- {
		args[i] = vm.pop()
	}
	recv := vm.pop() // This is self

	if recv == nil || recv.ComposedMethods == nil {
		return nil, &eval.Error{
			Kind:    "MessageNotUnderstood",
			Message: fmt.Sprintf("super dispatch failed: no composed method #%s", selector),
			Pos:     ast.Pos{Line: 1, Col: 1},
		}
	}

	if m, ok := recv.ComposedMethods[selector]; ok {
		return vm.applyMethod(recv, m, args)
	}

	return nil, &eval.Error{
		Kind:    "MessageNotUnderstood",
		Message: fmt.Sprintf("super dispatch failed: no composed method #%s", selector),
		Pos:     ast.Pos{Line: 1, Col: 1},
	}
}

// applyMethod invokes a method on self with the given arguments.
func (vm *VM) applyMethod(self *object.Object, m *object.MethodDef, args []*object.Object) (*object.Object, error) {
	// Native methods
	if m.Native != nil {
		return m.Native(self, args)
	}

	// For AST-based methods, we'd need to fall back to tree-walking
	// or have a pre-compiled version. For now, return nil.
	// In a full implementation, methods would be compiled to bytecode.
	return object.Nil, nil
}

// Stack operations

func (vm *VM) push(val *object.Object) {
	if vm.sp >= maxStackSize {
		panic("stack overflow")
	}
	vm.stack[vm.sp] = val
	vm.sp++
}

func (vm *VM) pop() *object.Object {
	if vm.sp <= 0 {
		panic("stack underflow")
	}
	vm.sp--
	return vm.stack[vm.sp]
}

func (vm *VM) peek(distance int) *object.Object {
	return vm.stack[vm.sp-1-distance]
}

// Global access

// SetGlobal sets a global variable.
func (vm *VM) SetGlobal(name string, val *object.Object) {
	vm.globals[name] = val
}

// GetGlobal gets a global variable.
func (vm *VM) GetGlobal(name string) (*object.Object, bool) {
	v, ok := vm.globals[name]
	return v, ok
}
