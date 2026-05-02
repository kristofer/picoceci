package module

import (
	"fmt"

	"github.com/kristofer/picoceci/pkg/ast"
	"github.com/kristofer/picoceci/pkg/bytecode"
	"github.com/kristofer/picoceci/pkg/lexer"
	"github.com/kristofer/picoceci/pkg/object"
	"github.com/kristofer/picoceci/pkg/parser"
)

// LoadedModule represents a compiled module.
type LoadedModule struct {
	Path    string                       // resolved path (for caching key)
	Chunk   *bytecode.Chunk              // compiled bytecode
	Blocks  []*bytecode.CompiledBlock    // compiled blocks for closures
	Globals map[string]*object.Object    // top-level object/interface declarations
}

// Loader manages module loading, compilation, caching, and circular import detection.
type Loader struct {
	resolver    *Resolver
	cache       map[string]*LoadedModule // resolved path -> loaded module
	importCache map[string]string        // import path -> resolved path (avoid re-resolving)
	loading     map[string]bool          // paths currently being loaded (cycle detection)
}

// NewLoader creates a new module loader with the given resolver.
func NewLoader(resolver *Resolver) *Loader {
	return &Loader{
		resolver:    resolver,
		cache:       make(map[string]*LoadedModule),
		importCache: make(map[string]string),
		loading:     make(map[string]bool),
	}
}

// Load loads a module by its import path.
// It returns a cached module if already loaded, detects circular imports,
// and compiles the module source if needed.
func (l *Loader) Load(importPath string) (*LoadedModule, error) {
	// Check if we've already resolved this import path
	if resolvedPath, ok := l.importCache[importPath]; ok {
		if mod, ok := l.cache[resolvedPath]; ok {
			return mod, nil
		}
	}

	// Resolve the path to get source and canonical path
	source, resolvedPath, err := l.resolver.Resolve(importPath)
	if err != nil {
		return nil, err
	}

	// Cache the import -> resolved mapping
	l.importCache[importPath] = resolvedPath

	// Check cache by resolved path
	if mod, ok := l.cache[resolvedPath]; ok {
		return mod, nil
	}

	// Check for circular import
	if l.loading[resolvedPath] {
		return nil, fmt.Errorf("circular import detected: %s", importPath)
	}

	// Mark as loading
	l.loading[resolvedPath] = true
	defer func() {
		delete(l.loading, resolvedPath)
	}()

	// Parse the source
	lex := lexer.NewString(source)
	p := parser.New(lex)
	program, parseErr := p.ParseProgram()
	if parseErr != nil {
		return nil, fmt.Errorf("parse error in %s: %v", importPath, parseErr)
	}

	// Extract top-level declarations (ObjectDecl, InterfaceDecl)
	globals := make(map[string]*object.Object)
	for _, stmt := range program.Statements {
		switch decl := stmt.(type) {
		case *ast.ObjectDecl:
			// Create a prototype object for the class
			// The name is stored for reference; actual methods are compiled separately
			obj := &object.Object{
				Kind:    object.KindObject,
				SVal:    decl.Name, // Store name in SVal for debugging/reference
				Methods: make(map[string]*object.MethodDef),
				Slots:   make(map[string]*object.Object),
			}
			globals[decl.Name] = obj

		case *ast.InterfaceDecl:
			// Create an interface object (placeholder for now)
			obj := &object.Object{
				Kind: object.KindObject,
				SVal: decl.Name,
			}
			globals[decl.Name] = obj

		case *ast.ImportDecl:
			// Handle nested imports recursively
			nestedMod, loadErr := l.Load(decl.Path)
			if loadErr != nil {
				return nil, fmt.Errorf("error loading %s from %s: %v", decl.Path, importPath, loadErr)
			}
			// Merge nested module's globals
			for name, obj := range nestedMod.Globals {
				globals[name] = obj
			}
		}
	}

	// Compile the module to bytecode
	compiler := bytecode.NewCompiler()
	chunk, compileErr := compiler.Compile(program.Statements)
	if compileErr != nil {
		return nil, fmt.Errorf("compile error in %s: %v", importPath, compileErr)
	}

	// Create the loaded module
	mod := &LoadedModule{
		Path:    resolvedPath,
		Chunk:   chunk,
		Blocks:  compiler.GetBlocks(),
		Globals: globals,
	}

	// Cache it
	l.cache[resolvedPath] = mod

	return mod, nil
}

// GetResolver returns the resolver used by this loader.
func (l *Loader) GetResolver() *Resolver {
	return l.resolver
}

// LoadModule implements bytecode.ModuleLoader interface.
// It loads a module and returns its globals and compiled blocks.
func (l *Loader) LoadModule(importPath string) (map[string]*object.Object, []*bytecode.CompiledBlock, error) {
	mod, err := l.Load(importPath)
	if err != nil {
		return nil, nil, err
	}
	return mod.Globals, mod.Blocks, nil
}

// LoadForEval implements eval.EvalModuleLoader interface.
// It loads a module and returns its globals for the tree-walking interpreter.
func (l *Loader) LoadForEval(importPath string) (map[string]*object.Object, error) {
	mod, err := l.Load(importPath)
	if err != nil {
		return nil, err
	}
	return mod.Globals, nil
}

// ClearCache clears the module cache.
func (l *Loader) ClearCache() {
	l.cache = make(map[string]*LoadedModule)
	l.importCache = make(map[string]string)
}

// IsCached returns true if the module at the given path is cached.
func (l *Loader) IsCached(importPath string) bool {
	// First check import cache
	if resolvedPath, ok := l.importCache[importPath]; ok {
		_, cached := l.cache[resolvedPath]
		return cached
	}
	return false
}
