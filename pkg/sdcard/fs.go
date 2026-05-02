package sdcard

import (
	"errors"
	"io/fs"
)

// OpenMode specifies how to open a file.
type OpenMode uint8

const (
	ModeRead      OpenMode = iota // Open for reading
	ModeWrite                     // Open for writing (truncate)
	ModeAppend                    // Open for appending
	ModeReadWrite                 // Open for read/write
)

// FileInfo represents file metadata.
type FileInfo interface {
	Name() string
	Size() int64
	IsDir() bool
}

// DirEntry represents a directory entry.
type DirEntry interface {
	Name() string
	IsDir() bool
}

// File represents an open file handle.
type File interface {
	// Read reads up to len(buf) bytes into buf.
	Read(buf []byte) (int, error)

	// ReadAll reads the entire file contents.
	ReadAll() ([]byte, error)

	// ReadLine reads a single line (without trailing newline).
	ReadLine() (string, error)

	// Write writes data to the file.
	Write(data []byte) (int, error)

	// Seek sets the offset for the next read/write.
	// whence: 0 = start, 1 = current, 2 = end
	Seek(offset int64, whence int) (int64, error)

	// Position returns the current file position.
	Position() int64

	// Size returns the file size in bytes.
	Size() int64

	// Flush writes any buffered data to storage.
	Flush() error

	// Close closes the file handle.
	Close() error

	// IsOpen returns true if the file is still open.
	IsOpen() bool
}

// FileSystem provides filesystem operations.
type FileSystem interface {
	// Open opens a file with the specified mode.
	Open(path string, mode OpenMode) (File, error)

	// Stat returns file info without opening.
	Stat(path string) (FileInfo, error)

	// Remove deletes a file.
	Remove(path string) error

	// Rename moves/renames a file.
	Rename(oldPath, newPath string) error

	// Mkdir creates a directory.
	Mkdir(path string) error

	// MkdirAll creates a directory and all parents.
	MkdirAll(path string) error

	// ReadDir lists directory contents.
	ReadDir(path string) ([]DirEntry, error)

	// RemoveAll removes a directory and all contents.
	RemoveAll(path string) error

	// Exists returns true if the path exists.
	Exists(path string) bool
}

var (
	// ErrNotMounted is returned when the filesystem is not mounted.
	ErrNotMounted = errors.New("sdcard: not mounted")

	// ErrNotFound is returned when a file/directory doesn't exist.
	ErrNotFound = fs.ErrNotExist

	// ErrIsDirectory is returned when a file operation is attempted on a directory.
	ErrIsDirectory = errors.New("sdcard: is a directory")

	// ErrNotDirectory is returned when a directory operation is attempted on a file.
	ErrNotDirectory = errors.New("sdcard: not a directory")

	// ErrPermission is returned for permission errors.
	ErrPermission = fs.ErrPermission
)

var (
	mounted   bool
	defaultFS FileSystem
)

// Mount initializes the filesystem at the given mount point.
// On desktop, this maps /sdcard/ to a local directory.
// On TinyGo, this initializes the SD card driver.
func Mount(mountPoint string) error {
	return mount(mountPoint)
}

// Unmount unmounts the filesystem.
func Unmount() error {
	mounted = false
	defaultFS = nil
	return nil
}

// IsMounted returns true if the filesystem is mounted.
func IsMounted() bool {
	return mounted
}

// FS returns the current filesystem.
// Returns nil if not mounted.
func FS() FileSystem {
	return defaultFS
}

// ReadFile is a convenience function that reads an entire file.
// This implements the module.FileReader interface.
func ReadFile(path string) ([]byte, error) {
	if defaultFS == nil {
		return nil, ErrNotMounted
	}
	f, err := defaultFS.Open(path, ModeRead)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.ReadAll()
}

// WriteFile is a convenience function that writes data to a file.
func WriteFile(path string, data []byte) error {
	if defaultFS == nil {
		return ErrNotMounted
	}
	f, err := defaultFS.Open(path, ModeWrite)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}
