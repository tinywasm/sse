//go:build !wasm

package tinysse

// New initializes a new TinySSE instance for the server.
func New(c *Config) *TinySSE {
	return &TinySSE{
		config: c,
		hub:    NewHub(c),
	}
}
