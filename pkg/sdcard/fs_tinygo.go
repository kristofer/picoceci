//go:build tinygo

package sdcard

import (
	"errors"
)

// mount initializes the SD card on TinyGo.
// This is a placeholder - actual implementation requires hardware.
func mount(mountPoint string) error {
	// TODO: Implement SD card initialization using:
	// - tinygo.org/x/drivers/sdcard
	// - tinygo.org/x/tinyfs/fatfs
	//
	// Example:
	//   dev := sdcard.New(machine.SDMMC{...})
	//   fs := fatfs.New(&dev)
	//   return fs.Mount()
	//
	// For now, return an error indicating SD card is not available.
	return errors.New("sdcard: TinyGo SD card driver not yet implemented")
}

// Note: The TinyGo implementation will need to implement:
// - tinygoFS struct implementing FileSystem
// - tinygoFile struct implementing File
// - Hardware initialization for ESP32-S3 SDMMC pins:
//   CLK: GPIO 39, CMD: GPIO 38, D0: GPIO 40
//   D1: GPIO 41, D2: GPIO 42, D3: GPIO 37
