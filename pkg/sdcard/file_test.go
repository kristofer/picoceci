package sdcard_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kristofer/picoceci/pkg/sdcard"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	sdcard.SetRoot(dir)
	if err := sdcard.Mount("/sdcard/"); err != nil {
		t.Fatalf("Mount failed: %v", err)
	}
	return dir
}

func TestFileWriteRead(t *testing.T) {
	setupTestDir(t)
	defer sdcard.Unmount()

	fs := sdcard.FS()

	// Write data
	f, err := fs.Open("/sdcard/test.txt", sdcard.ModeWrite)
	if err != nil {
		t.Fatalf("Open for write failed: %v", err)
	}
	data := []byte("Hello, picoceci!")
	n, err := f.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("wrote %d bytes, want %d", n, len(data))
	}
	f.Close()

	// Read back
	f2, err := fs.Open("/sdcard/test.txt", sdcard.ModeRead)
	if err != nil {
		t.Fatalf("Open for read failed: %v", err)
	}
	defer f2.Close()

	content, err := f2.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(content) != "Hello, picoceci!" {
		t.Errorf("got %q, want %q", string(content), "Hello, picoceci!")
	}
}

func TestFileAppend(t *testing.T) {
	setupTestDir(t)
	defer sdcard.Unmount()

	fs := sdcard.FS()

	// Initial write
	f1, _ := fs.Open("/sdcard/append.txt", sdcard.ModeWrite)
	f1.Write([]byte("line1\n"))
	f1.Close()

	// Append
	f2, _ := fs.Open("/sdcard/append.txt", sdcard.ModeAppend)
	f2.Write([]byte("line2\n"))
	f2.Close()

	// Read back
	f3, _ := fs.Open("/sdcard/append.txt", sdcard.ModeRead)
	content, _ := f3.ReadAll()
	f3.Close()

	expected := "line1\nline2\n"
	if string(content) != expected {
		t.Errorf("got %q, want %q", string(content), expected)
	}
}

func TestFileSeek(t *testing.T) {
	setupTestDir(t)
	defer sdcard.Unmount()

	fs := sdcard.FS()

	// Write data
	f, _ := fs.Open("/sdcard/seek.txt", sdcard.ModeWrite)
	f.Write([]byte("0123456789"))
	f.Close()

	// Open for read/write and test seek
	f2, _ := fs.Open("/sdcard/seek.txt", sdcard.ModeReadWrite)
	defer f2.Close()

	// Seek to position 5
	pos, err := f2.Seek(5, 0)
	if err != nil {
		t.Fatalf("Seek failed: %v", err)
	}
	if pos != 5 {
		t.Errorf("position = %d, want 5", pos)
	}

	// Read from position 5
	buf := make([]byte, 5)
	n, _ := f2.Read(buf)
	if n != 5 || string(buf) != "56789" {
		t.Errorf("got %q, want %q", string(buf[:n]), "56789")
	}
}

func TestFileReadLine(t *testing.T) {
	setupTestDir(t)
	defer sdcard.Unmount()

	fs := sdcard.FS()

	// Write multi-line file
	f, _ := fs.Open("/sdcard/lines.txt", sdcard.ModeWrite)
	f.Write([]byte("line1\nline2\nline3\n"))
	f.Close()

	// Read lines
	f2, _ := fs.Open("/sdcard/lines.txt", sdcard.ModeRead)
	defer f2.Close()

	expected := []string{"line1", "line2", "line3"}
	for i, exp := range expected {
		line, err := f2.ReadLine()
		if err != nil {
			t.Fatalf("ReadLine %d failed: %v", i, err)
		}
		if line != exp {
			t.Errorf("line %d: got %q, want %q", i, line, exp)
		}
	}
}

func TestFileSize(t *testing.T) {
	setupTestDir(t)
	defer sdcard.Unmount()

	fs := sdcard.FS()

	data := []byte("1234567890")
	f, _ := fs.Open("/sdcard/size.txt", sdcard.ModeWrite)
	f.Write(data)
	f.Close()

	f2, _ := fs.Open("/sdcard/size.txt", sdcard.ModeRead)
	defer f2.Close()

	if size := f2.Size(); size != 10 {
		t.Errorf("Size() = %d, want 10", size)
	}
}

func TestFilePosition(t *testing.T) {
	setupTestDir(t)
	defer sdcard.Unmount()

	fs := sdcard.FS()

	f, _ := fs.Open("/sdcard/pos.txt", sdcard.ModeWrite)
	f.Write([]byte("0123456789"))
	f.Close()

	f2, _ := fs.Open("/sdcard/pos.txt", sdcard.ModeRead)
	defer f2.Close()

	// Initial position
	if pos := f2.Position(); pos != 0 {
		t.Errorf("initial Position() = %d, want 0", pos)
	}

	// Read 5 bytes
	buf := make([]byte, 5)
	f2.Read(buf)

	// Position should be 5
	if pos := f2.Position(); pos != 5 {
		t.Errorf("after read Position() = %d, want 5", pos)
	}
}

func TestFileIsOpen(t *testing.T) {
	setupTestDir(t)
	defer sdcard.Unmount()

	fs := sdcard.FS()

	f, _ := fs.Open("/sdcard/open.txt", sdcard.ModeWrite)

	if !f.IsOpen() {
		t.Error("IsOpen() = false, want true")
	}

	f.Close()

	if f.IsOpen() {
		t.Error("IsOpen() = true after close, want false")
	}
}

func TestReadFile(t *testing.T) {
	dir := setupTestDir(t)
	defer sdcard.Unmount()

	// Create test file directly
	testPath := filepath.Join(dir, "readfile.txt")
	os.WriteFile(testPath, []byte("test content"), 0644)

	// Use ReadFile convenience function
	content, err := sdcard.ReadFile("/sdcard/readfile.txt")
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("got %q, want %q", string(content), "test content")
	}
}

func TestWriteFile(t *testing.T) {
	dir := setupTestDir(t)
	defer sdcard.Unmount()

	// Use WriteFile convenience function
	err := sdcard.WriteFile("/sdcard/writefile.txt", []byte("written content"))
	if err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Verify directly
	content, _ := os.ReadFile(filepath.Join(dir, "writefile.txt"))
	if string(content) != "written content" {
		t.Errorf("got %q, want %q", string(content), "written content")
	}
}

func TestMountUnmount(t *testing.T) {
	setupTestDir(t)

	if !sdcard.IsMounted() {
		t.Error("IsMounted() = false after Mount")
	}

	sdcard.Unmount()

	if sdcard.IsMounted() {
		t.Error("IsMounted() = true after Unmount")
	}

	if sdcard.FS() != nil {
		t.Error("FS() != nil after Unmount")
	}
}

func TestReadFileNotMounted(t *testing.T) {
	sdcard.Unmount()

	_, err := sdcard.ReadFile("/sdcard/test.txt")
	if err != sdcard.ErrNotMounted {
		t.Errorf("expected ErrNotMounted, got %v", err)
	}
}
