package tinysse

// TinySSE is the interface for the TinySSE service.
type TinySSE interface {
	// Broadcast sends a message to the hub.
	Broadcast(msg *SSEMessage)
}

// New creates a new TinySSE service.
// It will return a server or a client depending on the build target.
func New(cfg *Config) TinySSE {
	return newService(cfg)
}
