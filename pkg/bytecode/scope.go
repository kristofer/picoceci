package bytecode

// scope represents a lexical scope during compilation.
type scope struct {
	locals    []local   // local variables in this scope
	upvalues  []Upvalue // captured variables from enclosing scopes
	enclosing *scope    // parent scope
	depth     int       // nesting depth (0 = top-level)
}

// local represents a local variable in a scope.
type local struct {
	name     string
	depth    int  // scope depth where declared
	captured bool // true if captured by a nested block
}

// newScope creates a new scope.
func newScope(enclosing *scope) *scope {
	depth := 0
	if enclosing != nil {
		depth = enclosing.depth + 1
	}
	return &scope{
		locals:    make([]local, 0),
		upvalues:  make([]Upvalue, 0),
		enclosing: enclosing,
		depth:     depth,
	}
}

// declareLocal adds a local variable to the current scope.
// Returns the slot index.
func (s *scope) declareLocal(name string) int {
	s.locals = append(s.locals, local{
		name:  name,
		depth: s.depth,
	})
	return len(s.locals) - 1
}

// resolveLocal looks up a variable in the current scope's locals.
// Returns (slot, true) if found, or (-1, false) if not found.
func (s *scope) resolveLocal(name string) (int, bool) {
	// Search backwards (most recent first)
	for i := len(s.locals) - 1; i >= 0; i-- {
		if s.locals[i].name == name {
			return i, true
		}
	}
	return -1, false
}

// resolveUpvalue resolves a variable from an enclosing scope.
// Returns (upvalue index, true) if found, or (-1, false) if not found.
func (s *scope) resolveUpvalue(name string) (int, bool) {
	if s.enclosing == nil {
		return -1, false
	}

	// Check if it's a local in the immediately enclosing scope
	if slot, ok := s.enclosing.resolveLocal(name); ok {
		s.enclosing.locals[slot].captured = true
		return s.addUpvalue(uint8(slot), true), true
	}

	// Check if it's an upvalue in the enclosing scope (recursive)
	if upvalueIdx, ok := s.enclosing.resolveUpvalue(name); ok {
		return s.addUpvalue(uint8(upvalueIdx), false), true
	}

	return -1, false
}

// addUpvalue adds an upvalue descriptor, reusing existing if identical.
func (s *scope) addUpvalue(index uint8, isLocal bool) int {
	// Check if we already have this upvalue
	for i, uv := range s.upvalues {
		if uv.Index == index && uv.IsLocal == isLocal {
			return i
		}
	}
	s.upvalues = append(s.upvalues, Upvalue{Index: index, IsLocal: isLocal})
	return len(s.upvalues) - 1
}

// localCount returns the number of local variables in this scope.
func (s *scope) localCount() int {
	return len(s.locals)
}
