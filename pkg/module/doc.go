// Package module implements the picoceci module loader.
//
// The module system supports loading picoceci source files as modules,
// enabling multi-file programs via the import statement.
//
// # Resolution Order
//
// When resolving an import path, the resolver searches in this order:
//  1. Built-in modules (compiled into the runtime binary)
//  2. SD card path: /sdcard/picoceci/libs/<name>.pc
//  3. Absolute paths (if the import path starts with /)
//
// # Usage with Bytecode Compiler
//
//	resolver := module.NewResolver(os.ReadFile)
//	module.RegisterBuiltins(resolver)
//	loader := module.NewLoader(resolver)
//	compiler := bytecode.NewCompilerWithLoader(loader)
//
// # Usage with Tree-Walk Interpreter
//
//	resolver := module.NewResolver(os.ReadFile)
//	module.RegisterBuiltins(resolver)
//	loader := module.NewLoader(resolver)
//	interp := eval.NewWithLoader(loader)
//
// # Built-in Modules
//
// The following built-in modules are available via RegisterBuiltins():
//   - core: Basic extensions to native types
//   - io: I/O objects (Console, Transcript provided by runtime)
//   - collections: Collection classes (OrderedCollection, etc.)
//
// # Circular Import Detection
//
// The loader detects circular imports at load time and returns an error.
// Modules are cached by their resolved path to avoid recompilation.
//
// Phase 4 deliverable — see IMPLEMENTATION_PLAN.md.
package module
