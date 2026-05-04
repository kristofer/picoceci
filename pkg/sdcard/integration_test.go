//go:build !tinygo

package sdcard_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kristofer/picoceci/pkg/module"
	"github.com/kristofer/picoceci/pkg/sdcard"
)

func getSDCardRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	dir := wd
	for {
		if _, modErr := os.Stat(filepath.Join(dir, "go.mod")); modErr == nil {
			root := filepath.Join(dir, "testdata", "sdcard")
			if info, statErr := os.Stat(root); statErr == nil && info.IsDir() {
				return root
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	t.Fatalf("could not locate testdata/sdcard from %s", wd)
	return ""
}

func TestIntegrationModuleLoadFromSDCard(t *testing.T) {
	// Set up SD card stub
	sdcard.SetRoot(getSDCardRoot(t))
	if err := sdcard.Mount("/sdcard/"); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}
	defer sdcard.Unmount()

	// Create module loader with SD card file reader
	resolver := module.NewResolver(sdcard.ReadFile)
	module.RegisterBuiltins(resolver)
	loader := module.NewLoader(resolver)

	// Test that we can load a module from SD card path
	mod, err := loader.Load("/sdcard/picoceci/libs/SDCardModule.pc")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify the module was loaded and has the SDCardTest object declaration
	if mod.Globals == nil {
		t.Fatal("Module globals is nil")
	}

	obj, ok := mod.Globals["SDCardTest"]
	if !ok {
		t.Fatal("SDCardTest not found in module globals")
	}

	// Verify it's an object
	if obj == nil {
		t.Fatal("SDCardTest is nil")
	}

	t.Logf("Successfully loaded SDCardTest from SD card: %s", obj.PrintString())
}

func TestIntegrationReadDataFile(t *testing.T) {
	// Set up SD card stub
	sdcard.SetRoot(getSDCardRoot(t))
	if err := sdcard.Mount("/sdcard/"); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}
	defer sdcard.Unmount()

	// Test reading a data file
	data, err := sdcard.ReadFile("/sdcard/data/sample.txt")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	expected := "Hello from SD card!\nThis is line 2.\nThis is line 3.\n"
	if string(data) != expected {
		t.Errorf("ReadFile content = %q, want %q", string(data), expected)
	}
}

func TestIntegrationFileOperations(t *testing.T) {
	// Set up SD card stub
	sdcard.SetRoot(getSDCardRoot(t))
	if err := sdcard.Mount("/sdcard/"); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}
	defer sdcard.Unmount()

	fs := sdcard.FS()

	// Create a temporary test file
	testPath := "/sdcard/data/integration_test.txt"
	defer fs.Remove(testPath)

	// Write
	f, err := fs.Open(testPath, sdcard.ModeWrite)
	if err != nil {
		t.Fatalf("Open for write failed: %v", err)
	}
	_, err = f.Write([]byte("Integration test data"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	f.Close()

	// Read back
	f, err = fs.Open(testPath, sdcard.ModeRead)
	if err != nil {
		t.Fatalf("Open for read failed: %v", err)
	}
	data, err := f.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	f.Close()

	if string(data) != "Integration test data" {
		t.Errorf("Read data = %q, want %q", string(data), "Integration test data")
	}
}

func TestIntegrationDirectoryListing(t *testing.T) {
	// Set up SD card stub
	sdcard.SetRoot(getSDCardRoot(t))
	if err := sdcard.Mount("/sdcard/"); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}
	defer sdcard.Unmount()

	fs := sdcard.FS()

	// List directory
	entries, err := fs.ReadDir("/sdcard/picoceci/libs")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	// Should have at least SDCardModule.pc and MathUtils.pc
	if len(entries) < 2 {
		t.Errorf("ReadDir returned %d entries, want at least 2", len(entries))
	}

	// Check for expected files
	found := map[string]bool{}
	for _, e := range entries {
		found[e.Name()] = true
	}

	if !found["SDCardModule.pc"] {
		t.Error("SDCardModule.pc not found in directory listing")
	}
	if !found["MathUtils.pc"] {
		t.Error("MathUtils.pc not found in directory listing")
	}
}
