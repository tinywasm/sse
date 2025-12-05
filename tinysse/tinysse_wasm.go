//go:build wasm

package tinysse

import "fmt"

func newService(cfg *Config) TinySSE {
	return NewClient(cfg)
}

// Broadcast is a no-op on the client.
func (c *Client) Broadcast(msg *SSEMessage) {
	fmt.Println("Broadcast is a no-op on the client")
}
