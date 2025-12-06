package tinysse

// SSEMessage represents a message sent over SSE.
// Shared by both Server (for broadcasting) and Client (for consumption).
type SSEMessage struct {
	ID    string // SSE "id:" field - Required. Used for Last-Event-ID reconnection.
	Event string // SSE "event:" field - Optional. Allows routing to different handlers.
	Data  []byte // SSE "data:" field - RAW bytes, library does NOT parse.
}
