package module

import (
	"fmt"
	"testing"
)

// mockFileReader creates a file reader that returns predefined content.
func mockFileReader(files map[string]string) FileReader {
	return func(path string) ([]byte, error) {
		if content, ok := files[path]; ok {
			return []byte(content), nil
		}
		return nil, fmt.Errorf("file not found: %s", path)
	}
}

func TestResolverBuiltin(t *testing.T) {
	r := NewResolver(mockFileReader(nil))
	r.RegisterBuiltin("core", "| x | x := 42.")

	source, resolved, err := r.Resolve("core")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "| x | x := 42." {
		t.Errorf("expected source '| x | x := 42.', got %q", source)
	}
	if resolved != "builtin:core" {
		t.Errorf("expected resolved 'builtin:core', got %q", resolved)
	}
}

func TestResolverSDCard(t *testing.T) {
	files := map[string]string{
		"/sdcard/picoceci/libs/Counter.pc": "object Counter { }",
	}
	r := NewResolver(mockFileReader(files))

	source, resolved, err := r.Resolve("Counter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "object Counter { }" {
		t.Errorf("expected source 'object Counter { }', got %q", source)
	}
	if resolved != "/sdcard/picoceci/libs/Counter.pc" {
		t.Errorf("expected resolved path, got %q", resolved)
	}
}

func TestResolverAbsolutePath(t *testing.T) {
	files := map[string]string{
		"/home/user/MyModule.pc": "object MyModule { }",
	}
	r := NewResolver(mockFileReader(files))

	source, resolved, err := r.Resolve("/home/user/MyModule")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "object MyModule { }" {
		t.Errorf("expected source, got %q", source)
	}
	if resolved != "/home/user/MyModule.pc" {
		t.Errorf("expected resolved path, got %q", resolved)
	}
}

func TestResolverAbsolutePathWithExtension(t *testing.T) {
	files := map[string]string{
		"/home/user/MyModule.pc": "object MyModule { }",
	}
	r := NewResolver(mockFileReader(files))

	source, resolved, err := r.Resolve("/home/user/MyModule.pc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "object MyModule { }" {
		t.Errorf("expected source, got %q", source)
	}
	if resolved != "/home/user/MyModule.pc" {
		t.Errorf("expected resolved path, got %q", resolved)
	}
}

func TestResolverPriority(t *testing.T) {
	// Built-in should take priority over SD card
	files := map[string]string{
		"/sdcard/picoceci/libs/core.pc": "| sdcard |",
	}
	r := NewResolver(mockFileReader(files))
	r.RegisterBuiltin("core", "| builtin |")

	source, resolved, err := r.Resolve("core")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "| builtin |" {
		t.Errorf("expected built-in source, got %q", source)
	}
	if resolved != "builtin:core" {
		t.Errorf("expected builtin: prefix, got %q", resolved)
	}
}

func TestResolverNotFound(t *testing.T) {
	r := NewResolver(mockFileReader(nil))

	_, _, err := r.Resolve("NonExistent")
	if err == nil {
		t.Fatal("expected error for non-existent module")
	}
	expected := "IOError: module not found: NonExistent"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestResolverCustomSDCardRoot(t *testing.T) {
	files := map[string]string{
		"/custom/libs/Math.pc": "object Math { }",
	}
	r := NewResolver(mockFileReader(files))
	r.SetSDCardRoot("/custom/libs/")

	source, _, err := r.Resolve("Math")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "object Math { }" {
		t.Errorf("expected source, got %q", source)
	}
}

func TestIsBuiltin(t *testing.T) {
	r := NewResolver(mockFileReader(nil))
	r.RegisterBuiltin("io", "")

	if !r.IsBuiltin("io") {
		t.Error("expected io to be builtin")
	}
	if r.IsBuiltin("notbuiltin") {
		t.Error("expected notbuiltin to not be builtin")
	}
}

func TestListBuiltins(t *testing.T) {
	r := NewResolver(mockFileReader(nil))
	r.RegisterBuiltin("core", "")
	r.RegisterBuiltin("io", "")

	builtins := r.ListBuiltins()
	if len(builtins) != 2 {
		t.Errorf("expected 2 builtins, got %d", len(builtins))
	}
}
