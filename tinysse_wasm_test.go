//go:build wasm

package tinysse

import "testing"

func TestWASM_Connect(t *testing.T) {
	// This is a placeholder test.
	// In a real environment, this would require a browser and a running server.
	// For now, we'll just test that the code compiles.
	sse := New(&Config{})
	sse.Connect()
}
