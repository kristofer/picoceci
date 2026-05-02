package module

import (
	"fmt"
	"path/filepath"
	"strings"
)

// FileReader abstracts filesystem access for testing and platform portability.
type FileReader func(path string) ([]byte, error)

// Resolver resolves import paths to source code.
// It implements a three-tier search order:
//  1. Built-in modules (compiled into runtime)
//  2. SD card path (/sdcard/picoceci/libs/<name>.pc)
//  3. Absolute paths
type Resolver struct {
	builtins   map[string]string // name -> source code
	sdcardRoot string            // default: /sdcard/picoceci/libs/
	readFile   FileReader        // filesystem abstraction
}

// NewResolver creates a new module resolver.
// The readFile function is used for filesystem access (pass os.ReadFile for production,
// or a mock for testing).
func NewResolver(readFile FileReader) *Resolver {
	return &Resolver{
		builtins:   make(map[string]string),
		sdcardRoot: "/sdcard/picoceci/libs/",
		readFile:   readFile,
	}
}

// SetSDCardRoot sets the SD card library root path.
func (r *Resolver) SetSDCardRoot(root string) {
	r.sdcardRoot = root
}

// RegisterBuiltin registers a built-in module with its source code.
func (r *Resolver) RegisterBuiltin(name, source string) {
	r.builtins[name] = source
}

// Resolve resolves an import path to source code.
// Returns the source code, the resolved path (for caching), and any error.
//
// Resolution order:
//  1. Built-in modules (exact name match)
//  2. SD card: sdcardRoot + name + ".pc"
//  3. Absolute path (if starts with /)
//
// If the path is not found, returns an IOError.
func (r *Resolver) Resolve(importPath string) (source string, resolvedPath string, err error) {
	// 1. Check built-in modules first
	if src, ok := r.builtins[importPath]; ok {
		return src, "builtin:" + importPath, nil
	}

	// 2. Check SD card path (for non-absolute paths)
	if !strings.HasPrefix(importPath, "/") {
		sdPath := filepath.Join(r.sdcardRoot, importPath+".pc")
		if data, readErr := r.readFile(sdPath); readErr == nil {
			return string(data), sdPath, nil
		}
	}

	// 3. Check absolute path
	if strings.HasPrefix(importPath, "/") {
		// Try with .pc extension if not present
		absPath := importPath
		if !strings.HasSuffix(absPath, ".pc") {
			absPath = importPath + ".pc"
		}
		if data, readErr := r.readFile(absPath); readErr == nil {
			return string(data), absPath, nil
		}
		// Try without extension
		if data, readErr := r.readFile(importPath); readErr == nil {
			return string(data), importPath, nil
		}
	}

	return "", "", fmt.Errorf("IOError: module not found: %s", importPath)
}

// IsBuiltin returns true if the given module name is a built-in.
func (r *Resolver) IsBuiltin(name string) bool {
	_, ok := r.builtins[name]
	return ok
}

// ListBuiltins returns the names of all registered built-in modules.
func (r *Resolver) ListBuiltins() []string {
	names := make([]string, 0, len(r.builtins))
	for name := range r.builtins {
		names = append(names, name)
	}
	return names
}
