//go:build !wasm

package tinysse

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/cdvelop/tinystring"
)

// ServeHTTP implements the http.Handler interface.
func (s *TinySSE) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	origin := r.Header.Get("Origin")
	if len(s.config.AllowedOrigins) > 0 {
		allowed := false
		for _, o := range s.config.AllowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}
		if !allowed {
			err := &SSEError{Type: tinystring.Msg.Auth, Err: http.ErrNoCookie, Context: "origin not allowed"}
			if s.config.OnError != nil {
				s.config.OnError(err)
			}
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		err := &SSEError{Type: tinystring.Msg.Connect, Err: http.ErrNotSupported, Context: "streaming unsupported"}
		if s.config.OnError != nil {
			s.config.OnError(err)
		}
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// Extract token and validate
	token := r.URL.Query().Get("token")
	userID, role, err := s.config.TokenValidator(token)
	if err != nil {
		err := &SSEError{Type: tinystring.Msg.Auth, Err: err, Context: "token validation failed"}
		if s.config.OnError != nil {
			s.config.OnError(err)
		}
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Replay missed messages
	lastEventID := r.Header.Get("Last-Event-ID")
	if lastEventID != "" {
		messages := s.hub.GetMessagesSince(lastEventID)
		for _, msg := range messages {
			io.WriteString(w, tinystring.Fmt("id: %s\ndata: %s\n\n", msg.ID, msg.Data))
		}
		flusher.Flush()
	}

	// Create and register a new client
	clientID, err := randomHex(16)
	if err != nil {
		err := &SSEError{Type: tinystring.Msg.Connect, Err: err, Context: "client ID generation failed"}
		if s.config.OnError != nil {
			s.config.OnError(err)
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	client := &SSEClient{
		ID:       clientID,
		UserID:   userID,
		Role:     role,
		Channels: autoChannels(userID, role),
		Send:     make(chan SSEMessage, s.config.BufferSize),
	}
	s.hub.register(client)

	// Unregister client on disconnect
	defer s.hub.unregister(client)

	// Notify on connect
	if s.config.OnConnect != nil {
		s.config.OnConnect(client.ID)
	}

	// Notify on disconnect
	if s.config.OnDisconnect != nil {
		defer s.config.OnDisconnect(client.ID)
	}

	// Send messages to the client
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-client.Send:
			if !ok {
				return
			}
			io.WriteString(w, tinystring.Fmt("id: %s\ndata: %s\n\n", msg.ID, msg.Data))
			flusher.Flush()
		}
	}
}

// autoChannels generates the default channels for a user.
func autoChannels(userID, role string) []string {
	return []string{
		"all",
		"role:" + role,
		"user:" + userID,
	}
}

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
