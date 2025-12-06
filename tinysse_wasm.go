//go:build wasm

package tinysse

// New initializes a new TinySSE instance for WASM.
func New(c *Config) *TinySSE {
	return &TinySSE{
		config: c,
	}
}
