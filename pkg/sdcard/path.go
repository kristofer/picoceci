package sdcard

import (
	"path/filepath"
	"strings"
)

// Path provides platform-independent path manipulation.
// Paths are always POSIX-style (/-separated).
type Path struct {
	path string
}

// PathFrom creates a Path from a string.
func PathFrom(s string) *Path {
	return &Path{path: s}
}

// Basename returns the last element of the path.
// Example: "/sdcard/data/log.csv" -> "log.csv"
func (p *Path) Basename() string {
	return filepath.Base(p.path)
}

// Dirname returns the path's directory (parent).
// Example: "/sdcard/data/log.csv" -> "/sdcard/data"
func (p *Path) Dirname() *Path {
	return &Path{path: filepath.Dir(p.path)}
}

// Extension returns the file extension without the leading dot.
// Example: "log.csv" -> "csv"
func (p *Path) Extension() string {
	ext := filepath.Ext(p.path)
	return strings.TrimPrefix(ext, ".")
}

// Stem returns the filename without the extension.
// Example: "log.csv" -> "log"
func (p *Path) Stem() string {
	base := filepath.Base(p.path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

// Join appends a child path component.
// Example: "/sdcard/data".Join("log.csv") -> "/sdcard/data/log.csv"
func (p *Path) Join(child string) *Path {
	return &Path{path: filepath.Join(p.path, child)}
}

// WithSuffix appends a suffix to the path string.
// Example: "/sdcard/data/log".WithSuffix(".csv") -> "/sdcard/data/log.csv"
func (p *Path) WithSuffix(suffix string) *Path {
	return &Path{path: p.path + suffix}
}

// IsAbsolute returns true if the path is absolute.
func (p *Path) IsAbsolute() bool {
	return filepath.IsAbs(p.path) || strings.HasPrefix(p.path, "/")
}

// String returns the path as a string.
func (p *Path) String() string {
	return p.path
}

// Clean returns a cleaned version of the path.
func (p *Path) Clean() *Path {
	return &Path{path: filepath.Clean(p.path)}
}

// Parent is an alias for Dirname.
func (p *Path) Parent() *Path {
	return p.Dirname()
}

// Segments returns the path components.
// Example: "/sdcard/data/log.csv" -> ["sdcard", "data", "log.csv"]
func (p *Path) Segments() []string {
	clean := filepath.Clean(p.path)
	// Remove leading slash
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" || clean == "." {
		return nil
	}
	return strings.Split(clean, "/")
}
