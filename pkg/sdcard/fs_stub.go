//go:build !tinygo

package sdcard

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var sdcardRoot = "./testdata/sdcard"

// SetRoot sets the local directory that maps to /sdcard/.
// This is primarily for testing.
func SetRoot(root string) {
	sdcardRoot = root
}

// mount initializes the desktop filesystem stub.
func mount(mountPoint string) error {
	// Create the testdata directory if it doesn't exist
	if err := os.MkdirAll(sdcardRoot, 0755); err != nil {
		return err
	}
	defaultFS = &stubFS{}
	mounted = true
	return nil
}

// stubFS implements FileSystem using the local filesystem.
type stubFS struct{}

// mapPath converts a /sdcard/ path to the local filesystem path.
func mapPath(path string) string {
	// Handle /sdcard/ prefix
	if strings.HasPrefix(path, "/sdcard/") {
		return filepath.Join(sdcardRoot, strings.TrimPrefix(path, "/sdcard/"))
	}
	if strings.HasPrefix(path, "/sdcard") {
		return filepath.Join(sdcardRoot, strings.TrimPrefix(path, "/sdcard"))
	}
	// For absolute paths not under /sdcard, return as-is (for testing)
	return path
}

func (s *stubFS) Open(path string, mode OpenMode) (File, error) {
	realPath := mapPath(path)

	var flag int
	switch mode {
	case ModeRead:
		flag = os.O_RDONLY
	case ModeWrite:
		flag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	case ModeAppend:
		flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
	case ModeReadWrite:
		flag = os.O_RDWR | os.O_CREATE
	}

	// Ensure parent directory exists for write modes
	if mode != ModeRead {
		dir := filepath.Dir(realPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}

	f, err := os.OpenFile(realPath, flag, 0644)
	if err != nil {
		return nil, err
	}

	return &stubFile{
		file:   f,
		reader: bufio.NewReader(f),
		open:   true,
	}, nil
}

func (s *stubFS) Stat(path string) (FileInfo, error) {
	info, err := os.Stat(mapPath(path))
	if err != nil {
		return nil, err
	}
	return &stubFileInfo{info: info}, nil
}

func (s *stubFS) Remove(path string) error {
	return os.Remove(mapPath(path))
}

func (s *stubFS) Rename(oldPath, newPath string) error {
	return os.Rename(mapPath(oldPath), mapPath(newPath))
}

func (s *stubFS) Mkdir(path string) error {
	return os.Mkdir(mapPath(path), 0755)
}

func (s *stubFS) MkdirAll(path string) error {
	return os.MkdirAll(mapPath(path), 0755)
}

func (s *stubFS) ReadDir(path string) ([]DirEntry, error) {
	entries, err := os.ReadDir(mapPath(path))
	if err != nil {
		return nil, err
	}
	result := make([]DirEntry, len(entries))
	for i, e := range entries {
		result[i] = &stubDirEntry{entry: e}
	}
	return result, nil
}

func (s *stubFS) RemoveAll(path string) error {
	return os.RemoveAll(mapPath(path))
}

func (s *stubFS) Exists(path string) bool {
	_, err := os.Stat(mapPath(path))
	return err == nil
}

// stubFile implements File using os.File.
type stubFile struct {
	file   *os.File
	reader *bufio.Reader
	open   bool
}

func (f *stubFile) Read(buf []byte) (int, error) {
	return f.reader.Read(buf)
}

func (f *stubFile) ReadAll() ([]byte, error) {
	// Seek to start first
	if _, err := f.file.Seek(0, 0); err != nil {
		return nil, err
	}
	f.reader.Reset(f.file)
	return io.ReadAll(f.reader)
}

func (f *stubFile) ReadLine() (string, error) {
	line, err := f.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	// Strip trailing newline
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line, nil
}

func (f *stubFile) Write(data []byte) (int, error) {
	return f.file.Write(data)
}

func (f *stubFile) Seek(offset int64, whence int) (int64, error) {
	pos, err := f.file.Seek(offset, whence)
	if err != nil {
		return pos, err
	}
	// Reset the buffered reader after seek
	f.reader.Reset(f.file)
	return pos, nil
}

func (f *stubFile) Position() int64 {
	pos, _ := f.file.Seek(0, 1) // Seek 0 from current position
	// Account for buffered data that hasn't been consumed
	return pos - int64(f.reader.Buffered())
}

func (f *stubFile) Size() int64 {
	info, err := f.file.Stat()
	if err != nil {
		return 0
	}
	return info.Size()
}

func (f *stubFile) Flush() error {
	return f.file.Sync()
}

func (f *stubFile) Close() error {
	f.open = false
	return f.file.Close()
}

func (f *stubFile) IsOpen() bool {
	return f.open
}

// stubFileInfo implements FileInfo.
type stubFileInfo struct {
	info os.FileInfo
}

func (i *stubFileInfo) Name() string {
	return i.info.Name()
}

func (i *stubFileInfo) Size() int64 {
	return i.info.Size()
}

func (i *stubFileInfo) IsDir() bool {
	return i.info.IsDir()
}

// stubDirEntry implements DirEntry.
type stubDirEntry struct {
	entry os.DirEntry
}

func (e *stubDirEntry) Name() string {
	return e.entry.Name()
}

func (e *stubDirEntry) IsDir() bool {
	return e.entry.IsDir()
}
