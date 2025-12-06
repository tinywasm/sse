//go:build !wasm

package tinysse

import (
	"net/http"
)

// SSEServer handles Server-Sent Events HTTP connections.
type SSEServer struct {
	tinySSE *tinySSE
	config  *ServerConfig
	hub     *hub
}

// Server creates a new SSEServer instance.
func (t *tinySSE) Server(c *ServerConfig) *SSEServer {
	return &SSEServer{
		tinySSE: t,
		config:  c,
		hub:     newHub(t, c),
	}
}

// ServeHTTP implements the http.Handler interface.
func (s *SSEServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Resolve channels
	var channels []string
	var err error

	if s.config.ChannelProvider != nil {
		channels, err = s.config.ChannelProvider.ResolveChannels(r)
	} else {
		// Default behavior: reject if no provider configured
		http.Error(w, "channel provider not configured", http.StatusInternalServerError)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// 2. Set headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// 3. Register client
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// Flush headers immediately so client knows connection is open
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Create client connection
	client := &clientConnection{
		channels: channels,
		send:     make(chan []byte, s.config.ClientChannelBuffer),
	}

	// Handle Last-Event-ID for replay
	lastEventID := r.Header.Get("Last-Event-ID")

	s.hub.register <- registerRequest{
		client:      client,
		lastEventID: lastEventID,
	}

	// Ensure unregister on exit
	defer func() {
		s.hub.unregister <- client
	}()

	// 4. Loop to send messages
	for {
		select {
		case msg, ok := <-client.send:
			if !ok {
				return
			}
			_, err := w.Write(msg)
			if err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// Publish implements SSEPublisher.Publish
func (s *SSEServer) Publish(data []byte, channels ...string) {
	s.hub.broadcast <- &broadcastMessage{
		msg: &SSEMessage{
			Event: "", // Default
			Data:  data,
		},
		channels: channels,
	}
}

// PublishEvent implements SSEPublisher.PublishEvent
func (s *SSEServer) PublishEvent(event string, data []byte, channels ...string) {
	s.hub.broadcast <- &broadcastMessage{
		msg: &SSEMessage{
			Event: event,
			Data:  data,
		},
		channels: channels,
	}
}
