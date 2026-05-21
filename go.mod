module github.com/kristofer/picoceci

go 1.25.0

// picoceci — a small, high-protein Smalltalk-syntax Go-semantics language
// targeting TinyGo / ESP32-S3 with a single-runtime embedded architecture.
//
// Build for desktop:
//   go build ./...
//   go test ./...
//
// Build for ESP32-S3 (requires TinyGo 0.32+):
//   tinygo build -target=esp32-coreboard-v2 ./target/esp32s3

require (
	tinygo.org/x/drivers v0.35.0
	tinygo.org/x/tinyfs v0.5.0
)
