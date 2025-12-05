//go:build !wasm

package tinysse

func newService(cfg *Config) TinySSE {
	return NewServer(cfg)
}
