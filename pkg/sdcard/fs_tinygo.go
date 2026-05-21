//go:build tinygo

package sdcard

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"machine"

	driverssd "tinygo.org/x/drivers/sdcard"
	"tinygo.org/x/tinyfs"
	"tinygo.org/x/tinyfs/fatfs"
)

const (
	// GPIO wiring: SCK=14, MISO(DO)=2, MOSI(DI)=15, CS=13.
	sdSCK   = machine.GPIO14
	sdDOPin = machine.GPIO2
	sdDIPin = machine.GPIO15
	sdCS    = machine.GPIO13

	// Retry init in case card power-up timing is marginal.
	sdInitRetries    = 3
	sdInitRetryDelay = 150 * time.Millisecond
)

type tinygoFS struct {
	fs *fatfs.FATFS
}

// sdBlockDevice wraps driverssd.Device and implements tinyfs.BlockDevice by
// calling ReadData/WriteData directly.  This avoids the ReadAt address-doubling
// bug in the TinyGo sdcard driver for non-SDHC (SD v2, <2 GB) cards.
type sdBlockDevice struct {
	dev        *driverssd.Device
	baseSector uint32
}

func (d *sdBlockDevice) ReadAt(buf []byte, off int64) (int, error) {
	sector := d.baseSector + uint32(off/512)
	n := 0
	for len(buf) >= 512 {
		if err := d.dev.ReadData(sector, buf[:512]); err != nil {
			return n, err
		}
		buf = buf[512:]
		n += 512
		sector++
	}
	return n, nil
}

func (d *sdBlockDevice) WriteAt(buf []byte, off int64) (int, error) {
	sector := d.baseSector + uint32(off/512)
	n := 0
	for len(buf) >= 512 {
		if err := d.dev.WriteData(sector, buf[:512]); err != nil {
			return n, err
		}
		buf = buf[512:]
		n += 512
		sector++
	}
	return n, nil
}

func (d *sdBlockDevice) Size() int64 {
	sz := d.dev.Size() - int64(d.baseSector)*512
	if sz < 0 {
		return 0
	}
	return sz
}

func (d *sdBlockDevice) WriteBlockSize() int64                 { return 512 }
func (d *sdBlockDevice) EraseBlockSize() int64                 { return 512 }
func (d *sdBlockDevice) EraseBlocks(start, length int64) error { return nil }

type tinygoFile struct {
	file   tinyfs.File
	reader *bufio.Reader
	open   bool
	pos    int64
}

type tinygoFileInfo struct {
	info os.FileInfo
}

type tinygoDirEntry struct {
	info os.FileInfo
}

func (f *tinygoFileInfo) Name() string {
	return f.info.Name()
}

func (f *tinygoFileInfo) Size() int64 {
	return f.info.Size()
}

func (f *tinygoFileInfo) IsDir() bool {
	return f.info.IsDir()
}

func (d *tinygoDirEntry) Name() string {
	return d.info.Name()
}

func (d *tinygoDirEntry) IsDir() bool {
	return d.info.IsDir()
}

func firstPartitionStartLBA(sector0 []byte) (uint32, bool) {
	if len(sector0) < 512 {
		return 0, false
	}
	if sector0[510] != 0x55 || sector0[511] != 0xAA {
		return 0, false
	}
	for i := 0; i < 4; i++ {
		off := 446 + i*16
		partType := sector0[off+4]
		start := binary.LittleEndian.Uint32(sector0[off+8 : off+12])
		count := binary.LittleEndian.Uint32(sector0[off+12 : off+16])
		if partType == 0 || count == 0 {
			continue
		}
		if start > 0 {
			return start, true
		}
	}
	return 0, false
}

func normalizePath(path string) string {
	p := strings.TrimSpace(path)
	if p == "" || p == "/sdcard" || p == "/sdcard/" {
		return "/"
	}
	if strings.HasPrefix(p, "/sdcard/") {
		p = "/" + strings.TrimPrefix(p, "/sdcard/")
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

func joinPath(dir, name string) string {
	if dir == "/" {
		return "/" + name
	}
	return strings.TrimRight(dir, "/") + "/" + name
}

func openFlags(mode OpenMode) int {
	switch mode {
	case ModeRead:
		return os.O_RDONLY
	case ModeWrite:
		return os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	case ModeAppend:
		return os.O_WRONLY | os.O_CREATE | os.O_APPEND
	case ModeReadWrite:
		return os.O_RDWR | os.O_CREATE
	default:
		return os.O_RDONLY
	}
}

func (s *tinygoFS) Open(path string, mode OpenMode) (File, error) {
	norm := normalizePath(path)
	if mode == ModeRead {
		if info, err := s.fs.Stat(norm); err == nil && info.IsDir() {
			return nil, ErrIsDirectory
		}
	}
	f, err := s.fs.OpenFile(norm, openFlags(mode))
	if err != nil {
		return nil, err
	}
	tf := &tinygoFile{
		file:   f,
		reader: bufio.NewReader(f),
		open:   true,
	}
	if mode == ModeAppend {
		if info, statErr := f.Stat(); statErr == nil {
			tf.pos = info.Size()
		}
	}
	return tf, nil
}

func (s *tinygoFS) Stat(path string) (FileInfo, error) {
	info, err := s.fs.Stat(normalizePath(path))
	if err != nil {
		return nil, err
	}
	return &tinygoFileInfo{info: info}, nil
}

func (s *tinygoFS) Remove(path string) error {
	return s.fs.Remove(normalizePath(path))
}

func (s *tinygoFS) Rename(oldPath, newPath string) error {
	return s.fs.Rename(normalizePath(oldPath), normalizePath(newPath))
}

func (s *tinygoFS) Mkdir(path string) error {
	return s.fs.Mkdir(normalizePath(path), 0755)
}

func (s *tinygoFS) MkdirAll(path string) error {
	norm := normalizePath(path)
	if norm == "/" {
		return nil
	}
	parts := strings.Split(strings.TrimPrefix(norm, "/"), "/")
	cur := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		cur = joinPath(cur, part)
		if err := s.Mkdir(cur); err != nil {
			if _, statErr := s.Stat(cur); statErr != nil {
				return err
			}
		}
	}
	return nil
}

func (s *tinygoFS) ReadDir(path string) ([]DirEntry, error) {
	dir, err := s.fs.Open(normalizePath(path))
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	infos, err := dir.Readdir(0)
	if err != nil {
		return nil, err
	}
	entries := make([]DirEntry, len(infos))
	for i, info := range infos {
		entries[i] = &tinygoDirEntry{info: info}
	}
	return entries, nil
}

func (s *tinygoFS) RemoveAll(path string) error {
	norm := normalizePath(path)
	if norm == "/" {
		return nil
	}
	entries, err := s.ReadDir(norm)
	if err != nil {
		// Not a directory or doesn't exist; try plain remove.
		return s.Remove(norm)
	}
	for _, entry := range entries {
		child := joinPath(norm, entry.Name())
		if entry.IsDir() {
			if err := s.RemoveAll(child); err != nil {
				return err
			}
		} else {
			if err := s.Remove(child); err != nil {
				return err
			}
		}
	}
	return s.Remove(norm)
}

func (s *tinygoFS) Exists(path string) bool {
	_, err := s.Stat(path)
	return err == nil
}

func (f *tinygoFile) Read(buf []byte) (int, error) {
	n, err := f.reader.Read(buf)
	f.pos += int64(n)
	return n, err
}

func (f *tinygoFile) ReadAll() ([]byte, error) {
	if _, err := f.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	f.reader.Reset(f.file)
	data, err := io.ReadAll(f.reader)
	f.pos = int64(len(data))
	return data, err
}

func (f *tinygoFile) ReadLine() (string, error) {
	line, err := f.reader.ReadString('\n')
	f.pos += int64(len(line))
	if err != nil && err != io.EOF {
		return "", err
	}
	line = strings.TrimSuffix(line, "\n")
	line = strings.TrimSuffix(line, "\r")
	return line, nil
}

func (f *tinygoFile) Write(data []byte) (int, error) {
	n, err := f.file.Write(data)
	f.pos += int64(n)
	return n, err
}

func (f *tinygoFile) Seek(offset int64, whence int) (int64, error) {
	pos, err := f.file.Seek(offset, whence)
	if err != nil {
		return pos, err
	}
	f.pos = pos
	f.reader.Reset(f.file)
	return pos, nil
}

func (f *tinygoFile) Position() int64 {
	return f.pos
}

func (f *tinygoFile) Size() int64 {
	info, err := f.file.Stat()
	if err != nil {
		return 0
	}
	return info.Size()
}

func (f *tinygoFile) Flush() error {
	if syncer, ok := any(f.file).(interface{ Sync() error }); ok {
		return syncer.Sync()
	}
	return nil
}

func (f *tinygoFile) Close() error {
	f.open = false
	return f.file.Close()
}

func (f *tinygoFile) IsOpen() bool {
	return f.open
}

// mount initializes the SD card on TinyGo using SPI + FATFS.
func mount(mountPoint string) error {
	_ = mountPoint

	// Use SPI1 (hardware HSPI) only — both buses share the same GPIO pins
	// so probing SPI0 after SPI1 causes GPIO matrix contention and
	// non-deterministic reads.
	buses := []struct {
		name string
		spi  *machine.SPI
	}{
		{name: "SPI1", spi: machine.SPI1},
	}

	attemptErrors := make([]string, 0, len(buses)*sdInitRetries)
	for _, b := range buses {
		for attempt := 1; attempt <= sdInitRetries; attempt++ {
			sdCS.Configure(machine.PinConfig{Mode: machine.PinOutput})
			sdCS.High()

			dev := driverssd.New(b.spi, sdSCK, sdDIPin, sdDOPin, sdCS)
			if err := dev.Configure(); err != nil {
				attemptErrors = append(attemptErrors, fmt.Sprintf("%s direct configure failed (attempt %d/%d): %v", b.name, attempt, sdInitRetries, err))
				time.Sleep(sdInitRetryDelay)
				continue
			}
			// Do NOT reconfigure the SPI bus here — dev.Configure() already
			// set the bus to its post-init frequency.  A second Configure()
			// resets the ESP32-S3 SPI peripheral and corrupts subsequent reads.

			var sector0 [512]byte
			if err := dev.ReadData(0, sector0[:]); err != nil {
				attemptErrors = append(attemptErrors, fmt.Sprintf("%s direct read sector0 failed (attempt %d/%d): %v", b.name, attempt, sdInitRetries, err))
				time.Sleep(sdInitRetryDelay)
				continue
			}
			if sector0[510] != 0x55 || sector0[511] != 0xAA {
				attemptErrors = append(attemptErrors, fmt.Sprintf(
					"%s direct sector0 bad signature (attempt %d/%d): sig=%02X%02X first16=% X",
					b.name,
					attempt,
					sdInitRetries,
					sector0[510],
					sector0[511],
					sector0[:16],
				))
				time.Sleep(sdInitRetryDelay)
				continue
			}

			// Try raw mount (superfloppy FAT at sector 0).
			rawDev := &sdBlockDevice{dev: &dev, baseSector: 0}
			filesystem := fatfs.New(rawDev)
			filesystem.Configure(&fatfs.Config{SectorSize: 512})
			if err := filesystem.Mount(); err == nil {
				defaultFS = &tinygoFS{fs: filesystem}
				mounted = true
				return nil
			}

			// Try partition mount: find first FAT partition in MBR table.
			lba, ok := firstPartitionStartLBA(sector0[:])
			if !ok {
				attemptErrors = append(attemptErrors, fmt.Sprintf("%s raw mount failed, no partition found (attempt %d/%d): mbrSig=%02X%02X mbrPart0=% X", b.name, attempt, sdInitRetries, sector0[510], sector0[511], sector0[446:462]))
				time.Sleep(sdInitRetryDelay)
				continue
			}

			var partSector0 [512]byte
			if err := dev.ReadData(lba, partSector0[:]); err != nil {
				attemptErrors = append(attemptErrors, fmt.Sprintf("%s partition read failed (attempt %d/%d): LBA=%d err=%v", b.name, attempt, sdInitRetries, lba, err))
				time.Sleep(sdInitRetryDelay)
				continue
			}

			partDev := &sdBlockDevice{dev: &dev, baseSector: lba}
			partFS := fatfs.New(partDev)
			partFS.Configure(&fatfs.Config{SectorSize: 512})
			if partMountErr := partFS.Mount(); partMountErr == nil {
				defaultFS = &tinygoFS{fs: partFS}
				mounted = true
				return nil
			} else {
				attemptErrors = append(attemptErrors, fmt.Sprintf("%s partition@LBA%d mount failed (attempt %d/%d): %v partFirst16=% X partSig=%02X%02X",
					b.name, lba, attempt, sdInitRetries, partMountErr, partSector0[:16], partSector0[510], partSector0[511]))
			}
			time.Sleep(sdInitRetryDelay)
		}
	}

	if len(attemptErrors) > 0 {
		return fmt.Errorf("all SD init attempts failed: %s", strings.Join(attemptErrors, " | "))
	}
	return fmt.Errorf("sdcard: no usable SPI bus")
}
