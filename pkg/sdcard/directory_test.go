package sdcard_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/kristofer/picoceci/pkg/sdcard"
)

func TestDirectoryReadDir(t *testing.T) {
	dir := setupTestDir(t)
	defer sdcard.Unmount()

	// Create some test files
	os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("1"), 0644)
	os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("2"), 0644)
	os.Mkdir(filepath.Join(dir, "subdir"), 0755)

	fs := sdcard.FS()
	entries, err := fs.ReadDir("/sdcard/")
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	// Extract names and sort for comparison
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	sort.Strings(names)

	expected := []string{"file1.txt", "file2.txt", "subdir"}
	if len(names) != len(expected) {
		t.Fatalf("got %v, want %v", names, expected)
	}
	for i := range names {
		if names[i] != expected[i] {
			t.Errorf("entry %d: got %q, want %q", i, names[i], expected[i])
		}
	}
}

func TestDirectoryIsDir(t *testing.T) {
	dir := setupTestDir(t)
	defer sdcard.Unmount()

	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0644)
	os.Mkdir(filepath.Join(dir, "dir"), 0755)

	fs := sdcard.FS()
	entries, _ := fs.ReadDir("/sdcard/")

	var fileEntry, dirEntry sdcard.DirEntry
	for _, e := range entries {
		if e.Name() == "file.txt" {
			fileEntry = e
		}
		if e.Name() == "dir" {
			dirEntry = e
		}
	}

	if fileEntry.IsDir() {
		t.Error("file.txt IsDir() = true, want false")
	}
	if !dirEntry.IsDir() {
		t.Error("dir IsDir() = false, want true")
	}
}

func TestMkdir(t *testing.T) {
	dir := setupTestDir(t)
	defer sdcard.Unmount()

	fs := sdcard.FS()

	err := fs.Mkdir("/sdcard/newdir")
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(filepath.Join(dir, "newdir"))
	if err != nil {
		t.Fatalf("directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("created path is not a directory")
	}
}

func TestMkdirAll(t *testing.T) {
	dir := setupTestDir(t)
	defer sdcard.Unmount()

	fs := sdcard.FS()

	err := fs.MkdirAll("/sdcard/a/b/c")
	if err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	// Verify nested directories exist
	info, err := os.Stat(filepath.Join(dir, "a", "b", "c"))
	if err != nil {
		t.Fatalf("nested directories not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("created path is not a directory")
	}
}

func TestRemove(t *testing.T) {
	dir := setupTestDir(t)
	defer sdcard.Unmount()

	// Create a file
	testFile := filepath.Join(dir, "todelete.txt")
	os.WriteFile(testFile, []byte("x"), 0644)

	fs := sdcard.FS()
	err := fs.Remove("/sdcard/todelete.txt")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify file is gone
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("file still exists after Remove")
	}
}

func TestRemoveAll(t *testing.T) {
	dir := setupTestDir(t)
	defer sdcard.Unmount()

	// Create nested structure
	os.MkdirAll(filepath.Join(dir, "tree", "a", "b"), 0755)
	os.WriteFile(filepath.Join(dir, "tree", "file.txt"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(dir, "tree", "a", "file.txt"), []byte("x"), 0644)

	fs := sdcard.FS()
	err := fs.RemoveAll("/sdcard/tree")
	if err != nil {
		t.Fatalf("RemoveAll failed: %v", err)
	}

	// Verify directory is gone
	if _, err := os.Stat(filepath.Join(dir, "tree")); !os.IsNotExist(err) {
		t.Error("directory still exists after RemoveAll")
	}
}

func TestRename(t *testing.T) {
	dir := setupTestDir(t)
	defer sdcard.Unmount()

	// Create a file
	os.WriteFile(filepath.Join(dir, "old.txt"), []byte("content"), 0644)

	fs := sdcard.FS()
	err := fs.Rename("/sdcard/old.txt", "/sdcard/new.txt")
	if err != nil {
		t.Fatalf("Rename failed: %v", err)
	}

	// Verify old is gone
	if _, err := os.Stat(filepath.Join(dir, "old.txt")); !os.IsNotExist(err) {
		t.Error("old file still exists after Rename")
	}

	// Verify new exists with correct content
	content, err := os.ReadFile(filepath.Join(dir, "new.txt"))
	if err != nil {
		t.Fatalf("new file not found: %v", err)
	}
	if string(content) != "content" {
		t.Errorf("got %q, want %q", string(content), "content")
	}
}

func TestStat(t *testing.T) {
	dir := setupTestDir(t)
	defer sdcard.Unmount()

	os.WriteFile(filepath.Join(dir, "stat.txt"), []byte("12345"), 0644)

	fs := sdcard.FS()
	info, err := fs.Stat("/sdcard/stat.txt")
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	if info.Name() != "stat.txt" {
		t.Errorf("Name() = %q, want %q", info.Name(), "stat.txt")
	}
	if info.Size() != 5 {
		t.Errorf("Size() = %d, want 5", info.Size())
	}
	if info.IsDir() {
		t.Error("IsDir() = true, want false")
	}
}

func TestExists(t *testing.T) {
	dir := setupTestDir(t)
	defer sdcard.Unmount()

	os.WriteFile(filepath.Join(dir, "exists.txt"), []byte("x"), 0644)

	fs := sdcard.FS()

	if !fs.Exists("/sdcard/exists.txt") {
		t.Error("Exists() = false for existing file")
	}
	if fs.Exists("/sdcard/notexists.txt") {
		t.Error("Exists() = true for non-existing file")
	}
}

func TestCreateParentDirs(t *testing.T) {
	setupTestDir(t)
	defer sdcard.Unmount()

	fs := sdcard.FS()

	// Opening a file for write should create parent directories
	f, err := fs.Open("/sdcard/new/nested/file.txt", sdcard.ModeWrite)
	if err != nil {
		t.Fatalf("Open with nested path failed: %v", err)
	}
	f.Write([]byte("test"))
	f.Close()

	// Verify file exists
	if !fs.Exists("/sdcard/new/nested/file.txt") {
		t.Error("nested file was not created")
	}
}
