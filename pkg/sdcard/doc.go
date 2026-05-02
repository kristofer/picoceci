// Package sdcard provides the picoceci SD card / filesystem objects.
//
// See docs/sdcard.md for the full API specification.
//
// # Usage
//
// Mount the filesystem before use:
//
//	sdcard.Mount("/sdcard/")
//	defer sdcard.Unmount()
//
//	// Read a file
//	content, _ := sdcard.ReadFile("/sdcard/data.txt")
//
//	// Write a file
//	sdcard.WriteFile("/sdcard/output.txt", []byte("hello"))
//
//	// Full file operations
//	f, _ := sdcard.FS().Open("/sdcard/log.txt", sdcard.ModeAppend)
//	f.Write([]byte("log entry\n"))
//	f.Close()
//
// # Path Manipulation
//
// The Path type provides platform-independent path operations:
//
//	p := sdcard.PathFrom("/sdcard/data/log.csv")
//	p.Basename()   // "log.csv"
//	p.Dirname()    // "/sdcard/data"
//	p.Extension()  // "csv"
//
// # Build Tags
//
//   - tinygo  — uses tinygo.org/x/drivers/sdcard + fatfs
//   - !tinygo — maps /sdcard/ paths to ./testdata/sdcard/ for desktop testing
//
// Phase 5 deliverable — see IMPLEMENTATION_PLAN.md.
package sdcard
