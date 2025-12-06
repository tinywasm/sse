//go:build wasm

package tinysse

// ClientConfig holds configuration strictly for the Browser/WASM Client.
type ClientConfig struct {
	// Endpoint is the SSE server URL.
	Endpoint string

	// RetryInterval in milliseconds for reconnection.
	RetryInterval int

	// MaxRetryDelay caps the exponential backoff.
	MaxRetryDelay int

	// MaxReconnectAttempts limits retry attempts. 0 = unlimited.
	MaxReconnectAttempts int
}
