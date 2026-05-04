package module

import (
	"strings"
	"testing"
)

func TestLoaderSimpleModule(t *testing.T) {
	files := map[string]string{
		"/sdcard/picoceci/libs/Math.pc": `
| x: Any |
x := 42.
`,
	}
	resolver := NewResolver(mockFileReader(files))
	loader := NewLoader(resolver)

	mod, err := loader.Load("Math")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mod == nil {
		t.Fatal("expected module, got nil")
	}
	if mod.Path != "/sdcard/picoceci/libs/Math.pc" {
		t.Errorf("unexpected path: %s", mod.Path)
	}
	if mod.Chunk == nil {
		t.Error("expected compiled chunk")
	}
}

func TestLoaderCaching(t *testing.T) {
	loadCount := 0
	files := map[string]string{
		"/sdcard/picoceci/libs/Counter.pc": "| x: Any | x := 0.",
	}
	countingReader := func(path string) ([]byte, error) {
		loadCount++
		if content, ok := files[path]; ok {
			return []byte(content), nil
		}
		return nil, &testError{"file not found"}
	}

	resolver := NewResolver(countingReader)
	loader := NewLoader(resolver)

	// First load
	_, err := loader.Load("Counter")
	if err != nil {
		t.Fatalf("first load error: %v", err)
	}
	if loadCount != 1 {
		t.Errorf("expected 1 file read, got %d", loadCount)
	}

	// Second load should use cache (no new file reads)
	beforeSecondLoad := loadCount
	_, err = loader.Load("Counter")
	if err != nil {
		t.Fatalf("second load error: %v", err)
	}
	if loadCount != beforeSecondLoad {
		t.Errorf("expected no new file reads on cached load, got %d more", loadCount-beforeSecondLoad)
	}
}

func TestLoaderCircularImport(t *testing.T) {
	files := map[string]string{
		"/sdcard/picoceci/libs/A.pc": "import 'B'.",
		"/sdcard/picoceci/libs/B.pc": "import 'A'.",
	}
	resolver := NewResolver(mockFileReader(files))
	loader := NewLoader(resolver)

	_, err := loader.Load("A")
	if err == nil {
		t.Fatal("expected circular import error")
	}
	if !strings.Contains(err.Error(), "circular import") {
		t.Errorf("expected 'circular import' in error, got: %v", err)
	}
}

func TestLoaderMissingModule(t *testing.T) {
	resolver := NewResolver(mockFileReader(nil))
	loader := NewLoader(resolver)

	_, err := loader.Load("NonExistent")
	if err == nil {
		t.Fatal("expected error for missing module")
	}
	if !strings.Contains(err.Error(), "IOError") {
		t.Errorf("expected IOError, got: %v", err)
	}
}

func TestLoaderParseError(t *testing.T) {
	files := map[string]string{
		"/sdcard/picoceci/libs/Bad.pc": "this is not valid picoceci {{{{",
	}
	resolver := NewResolver(mockFileReader(files))
	loader := NewLoader(resolver)

	_, err := loader.Load("Bad")
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "parse error") {
		t.Errorf("expected 'parse error' in message, got: %v", err)
	}
}

func TestLoaderBuiltinModule(t *testing.T) {
	resolver := NewResolver(mockFileReader(nil))
	resolver.RegisterBuiltin("core", "| x: Bool | x := true.")
	loader := NewLoader(resolver)

	mod, err := loader.Load("core")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mod.Path != "builtin:core" {
		t.Errorf("expected builtin path, got: %s", mod.Path)
	}
}

func TestLoaderNestedImport(t *testing.T) {
	files := map[string]string{
		"/sdcard/picoceci/libs/Main.pc":   "import 'Helper'. | x: Int | x := 1.",
		"/sdcard/picoceci/libs/Helper.pc": "| y: Int | y := 2.",
	}
	resolver := NewResolver(mockFileReader(files))
	loader := NewLoader(resolver)

	mod, err := loader.Load("Main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mod == nil {
		t.Fatal("expected module")
	}
	// Both modules should be cached
	if !loader.IsCached("Main") {
		t.Error("Main should be cached")
	}
	if !loader.IsCached("Helper") {
		t.Error("Helper should be cached")
	}
}

func TestLoaderClearCache(t *testing.T) {
	files := map[string]string{
		"/sdcard/picoceci/libs/Test.pc": "| x: Int | x := 1.",
	}
	resolver := NewResolver(mockFileReader(files))
	loader := NewLoader(resolver)

	_, _ = loader.Load("Test")
	if !loader.IsCached("Test") {
		t.Error("expected Test to be cached")
	}

	loader.ClearCache()
	if loader.IsCached("Test") {
		t.Error("expected cache to be cleared")
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
