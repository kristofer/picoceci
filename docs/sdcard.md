# SD Card / Filesystem API

Version: 0.1-draft  
Audience: Implementors of `pkg/sdcard/`

---

## Overview

picoceci treats the SD card as its primary persistent storage.  On the ESP32-S3-N16R8 a microSD card (SDMMC or SPI mode) is mounted as a FAT32 (or littlefs) volume at `/sdcard/`.  The filesystem objects exposed to picoceci programs are a thin wrapper over TinyGo's `machine/sdcard` and Go-compatible `os`-like calls.

---

## Hardware Setup

### Recommended wiring (ESP32-S3, SDMMC 4-bit)

| Signal | ESP32-S3 GPIO |
|---|---|
| CLK | GPIO 39 |
| CMD (MOSI) | GPIO 38 |
| D0 (MISO) | GPIO 40 |
| D1 | GPIO 41 |
| D2 | GPIO 42 |
| D3 (CS) | GPIO 37 |
| Detect (optional) | GPIO 21 |

4-bit SDMMC mode provides ~20 MB/s read speed on a Class 10 card.

### SPI fallback (for boards without SDMMC)

```picoceci
SDCard mountSPI: (SPI new: 2 sck: 14 mosi: 13 miso: 12 cs: 15 speed: 20000000).
```

---

## Initialization

### Auto-mount (default)

The runtime entry point (`target/esp32s3/main.go`) calls `sdcard.Mount("/sdcard/")` before starting the interpreter.  If mounting fails, picoceci starts without SD card support; `IOError` is raised on any filesystem operation.

### Manual mount from picoceci

```picoceci
SDCard mount: '/sdcard/'.
SDCard mounted.    "=> true"
SDCard totalBytes. "=> total capacity in bytes"
SDCard freeBytes.  "=> available bytes"
```

---

## `File` Object

### Opening

```picoceci
| f |
f := File open: '/sdcard/data.txt' mode: #read.
f := File open: '/sdcard/log.txt' mode: #write.    "create or truncate"
f := File open: '/sdcard/log.txt' mode: #append.   "create or append"
f := File open: '/sdcard/db.bin'  mode: #readWrite.
```

### Reading

```picoceci
| bytes line |

"Read all bytes"
bytes := f readAll.

"Read n bytes"
bytes := f read: 512.

"Read one line"
line := f readLine.

"Read into existing buffer"
| buf count |
buf := ByteArray new: 1024.
count := f readInto: buf count: 1024.
```

### Writing

```picoceci
f write: 'Hello' asBytes.
f write: #[ 1 2 3 255 ].
f flush.
```

### Seeking

```picoceci
f position.              "=> current byte offset"
f position: 0.           "seek to start"
f position: f size - 4.  "seek near end"
f seekToEnd.
```

### Closing

```picoceci
f close.
```

### `File` convenience class methods

```picoceci
"Read entire file to String"
| text |
text := File read: '/sdcard/readme.txt'.

"Write String to file (overwrite)"
File write: 'Hello' to: '/sdcard/out.txt'.

"Append String to file"
File append: 'more\n' to: '/sdcard/log.txt'.

"Check existence"
File exists: '/sdcard/data.csv'.   "=> true or false"

"Delete"
File delete: '/sdcard/tmp.bin'.

"Copy"
File copy: '/sdcard/a.txt' to: '/sdcard/b.txt'.

"Move / rename"
File move: '/sdcard/old.txt' to: '/sdcard/new.txt'.

"File size"
File size: '/sdcard/data.bin'.   "=> Integer bytes"
```

### Full `File` message table

| Message | Return | Description |
|---|---|---|
| `File open: path mode: #read` | File | Open for reading |
| `File open: path mode: #write` | File | Open for writing (truncate) |
| `File open: path mode: #append` | File | Open for appending |
| `File open: path mode: #readWrite` | File | Open for read/write |
| `File read: path` | String | Read entire file |
| `File write: str to: path` | self | Write string to file |
| `File append: str to: path` | self | Append string to file |
| `File exists: path` | Boolean | Existence check |
| `File delete: path` | self | Delete file |
| `File copy: from to: to` | self | Copy file |
| `File move: from to: to` | self | Move / rename file |
| `File size: path` | Integer | File size in bytes |
| `f read: n` | ByteArray | Read n bytes |
| `f readAll` | ByteArray | Read all bytes |
| `f readLine` | String | Read line (strips newline) |
| `f readInto: buf count: n` | Integer | Read into buffer, return count |
| `f write: byteArray` | Integer | Write bytes, return count written |
| `f flush` | self | Flush write buffers |
| `f position` | Integer | Current offset |
| `f position: n` | self | Seek to absolute offset |
| `f seekToEnd` | self | Seek to end |
| `f size` | Integer | File size in bytes |
| `f close` | self | Close file handle |
| `f isOpen` | Boolean | Open state |

---

## `Directory` Object

```picoceci
| dir entries |

"List entries"
dir := Directory open: '/sdcard/'.
entries := dir entries.   "=> Array of Strings (names)"
dir close.

"Class-level convenience"
Directory entries: '/sdcard/'.
Directory exists: '/sdcard/mydir'.
Directory create: '/sdcard/newdir'.
Directory delete: '/sdcard/emptydir'.
Directory deleteAll: '/sdcard/tree'.   "recursive"
```

### Full `Directory` message table

| Message | Return | Description |
|---|---|---|
| `Directory open: path` | Directory | Open directory |
| `Directory entries: path` | Array of String | List names |
| `Directory exists: path` | Boolean | Existence check |
| `Directory create: path` | self | Create directory |
| `Directory createAll: path` | self | Create with parents |
| `Directory delete: path` | self | Remove empty directory |
| `Directory deleteAll: path` | self | Recursive delete |
| `d entries` | Array of String | Names in open directory |
| `d close` | self | Release handle |
| `d do: aBlock` | self | Iterate over names |

---

## `Path` Object

`Path` provides platform-independent path manipulation.  Paths are always POSIX-style (`/`-separated).

```picoceci
| p |
p := Path from: '/sdcard/data/log.csv'.

p basename.    "=> 'log.csv'"
p dirname.     "=> '/sdcard/data'"
p extension.   "=> 'csv'"
p stem.        "=> 'log'"

p , 'sibling.txt'.   "=> '/sdcard/data/sibling.txt'"
p / 'child.txt'.     "=> error (p is a file path)"

| dir |
dir := Path from: '/sdcard/data'.
dir / 'log.csv'.   "=> '/sdcard/data/log.csv'"
```

| Message | Return | Description |
|---|---|---|
| `Path from: string` | Path | Construct from string |
| `p basename` | String | Last path component |
| `p dirname` | Path | Parent directory |
| `p extension` | String | File extension (no dot) |
| `p stem` | String | Basename without extension |
| `p / name` | Path | Append child component |
| `p , suffix` | Path | Append suffix string |
| `p asString` | String | Canonical string |
| `p isAbsolute` | Boolean | Absolute path check |

---

## Streams over Files

Streams provide higher-level buffered access:

```picoceci
| stream |
stream := ReadStream on: (File open: '/sdcard/data.csv' mode: #read).

[ stream atEnd ] whileFalse: [
    | line |
    line := stream nextLine.
    Console println: line
].
stream close.
```

```picoceci
| out |
out := WriteStream on: (File open: '/sdcard/out.txt' mode: #write).
out nextPutAll: 'hello\n'.
out nextPutAll: 'world\n'.
out close.
```

---

## Error Handling

All SD card operations raise `IOError` on failure:

```picoceci
[ File read: '/sdcard/noexist.txt' ]
    on: IOError
    do: [ :e | Console println: 'File not found: ', e messageText ].
```

`IOError` carries:

| Slot | Content |
|---|---|
| `messageText` | Human-readable description |
| `errno` | Underlying POSIX errno code |
| `path` | The path that caused the error |

---

## Implementation Notes

### TinyGo target

Use `tinygo.org/x/drivers/sdcard` for SDMMC/SPI SD card driver.  Mount with `fatfs` (TinyGo's FAT32 binding):

```go
//go:build tinygo

import (
    "tinygo.org/x/drivers/sdcard"
    "tinygo.org/x/tinyfs/fatfs"
)

func Mount(mountPoint string) error {
    dev := sdcard.New(machine.SDMMC{...})
    fs := fatfs.New(&dev)
    return fs.Mount()
}
```

### Desktop stub

On desktop, all paths under `/sdcard/` are mapped to a configurable local directory (default `./testdata/sdcard/`).  This allows unit testing without hardware.

```go
//go:build !tinygo

var sdcardRoot = "./testdata/sdcard"

func Mount(mountPoint string) error {
    // create testdata/sdcard if it doesn't exist
    return os.MkdirAll(sdcardRoot, 0755)
}
```

### Write buffering

Write operations are buffered in 512-byte sectors (SD card native block size).  `flush` forces the buffer to disk.  File `close` implies `flush`.

### FAT32 limitations

| Limit | Value |
|---|---|
| Max filename length | 255 characters (LFN) |
| Max file size | 4 GB - 1 byte |
| Max volume size | 32 GB (FAT32 standard) |
| Max open files simultaneously | Configurable (default 8) |

For volumes > 32 GB, use exFAT (requires `tinygo.org/x/tinyfs/exfatfs` when available) or littlefs.

---

## SD Card Performance Tips

- Use 4-bit SDMMC mode for maximum throughput (~20 MB/s).
- Allocate read/write buffers in PSRAM to leave internal SRAM for ISR stacks.
- Batch small writes into a `WriteStream` and flush once; FAT32 writes are expensive at sector boundaries.
- For log files, use append mode and a ring-buffer pattern to limit wear.
- Consider `littlefs` for write-heavy workloads (better wear levelling than FAT32).

---

*End of sdcard.md*
