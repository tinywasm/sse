//go:build !wasm

package tinysse

import (
	"crypto/rand"
	"fmt"
	"net/http"
)

// Server is the SSE server.
type Server struct {
	hub *SSEHub
	cfg *Config
}

// NewServer creates a new SSE server.
func NewServer(cfg *Config) *Server {
	return &Server{
		hub: newHub(cfg.BufferSize),
		cfg: cfg,
	}
}

// ServeHTTP handles HTTP requests for the SSE server.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	if len(s.cfg.AllowedOrigins) > 0 {
		for _, origin := range s.cfg.AllowedOrigins {
			if r.Header.Get("Origin") == origin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				break
			}
		}
	}

	// Get token from query param
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token is required", http.StatusUnauthorized)
		return
	}

	// Validate token
	userID, role, err := s.cfg.TokenValidator(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Create a new client ID
	b := make([]byte, 16)
	_, err = rand.Read(b)
	if err != nil {
		http.Error(w, "could not generate client ID", http.StatusInternalServerError)
		return
	}
	clientID := fmt.Sprintf("%x", b)

	// Create a new client
	client := &SSEClient{
		ID:       clientID,
		UserID:   userID,
		Role:     role,
		Channels: autoChannels(userID, role),
		Send:     make(chan []byte, s.cfg.BufferSize),
	}

	// Add client to the hub
	s.hub.add(client)
	if s.cfg.OnConnect != nil {
		s.cfg.OnConnect(client.ID)
	}

	// Remove client on disconnect
	defer func() {
		s.hub.remove(client.ID)
		if s.cfg.OnDisconnect != nil {
			s.cfg.OnDisconnect(client.ID)
		}
	}()

	// Get last event ID
	lastEventID := r.Header.Get("Last-Event-ID")
	if lastEventID == "" {
		lastEventID = r.URL.Query().Get("lastEventId")
	}

	// Send missed messages
	if lastEventID != "" {
		s.hub.mu.RLock()
		for _, msg := range s.hub.buffer {
			if msg.ID > lastEventID {
				fmt.Fprintf(w, "id: %s\ndata: %s\n\n", msg.ID, msg.Data)
			}
		}
		s.hub.mu.RUnlock()
	}

	// Listen for messages to send
	for {
		select {
		case msg, ok := <-client.Send:
			if !ok {
				return // Channel closed
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-r.Context().Done():
			return // Client disconnected
		}
	}
}

// Broadcast sends a message to the hub.
func (s *Server) Broadcast(msg *SSEMessage) {
	s.hub.broadcast(msg)
}
