package main
//go:build tinygo && !esp32s3_idf_bridge

package main

func runtimeNetworkMode() string {
	return "tinygo-default"
}
