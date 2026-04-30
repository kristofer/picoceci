module github.com/kristofer/picoceci

go 1.22

// picoceci — a small, high-protein Smalltalk-syntax Go-semantics language
// targeting TinyGo / ESP32-S3 via the Canal capability microkernel.
//
// Build for desktop:
//   go build ./...
//   go test ./...
//
// Build for ESP32-S3 (requires TinyGo 0.32+):
//   tinygo build -target=esp32-coreboard-v2 ./target/esp32s3
