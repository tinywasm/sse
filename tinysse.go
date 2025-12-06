package tinysse

// TinySSE is the main struct for the library.
type TinySSE struct {
	config *Config
	hub    *SSEHub // hub is server-side only
}

// Broadcast sends a message to the specified channels.
func (s *TinySSE) Broadcast(data []byte, broadcast []string, handlerID uint8) {
	s.hub.Broadcast(data, broadcast, handlerID)
}
