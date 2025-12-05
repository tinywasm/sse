//go:build !wasm

package tinysse

import (
	"sync"
)

// SSEHub manages SSE clients and message broadcasting.
type SSEHub struct {
	mu      sync.RWMutex
	clients map[string]*SSEClient
	buffer  []*SSEMessage
	// BufferSize is the maximum number of messages to buffer.
	BufferSize int
}

// SSEClient represents a connected SSE client.
type SSEClient struct {
	ID       string
	UserID   string
	Role     string
	Channels []string
	Send     chan []byte
}

// newHub creates a new SSEHub.
func newHub(bufferSize int) *SSEHub {
	return &SSEHub{
		clients:    make(map[string]*SSEClient),
		buffer:     make([]*SSEMessage, 0, bufferSize),
		BufferSize: bufferSize,
	}
}

// add adds a client to the hub.
func (h *SSEHub) add(client *SSEClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client.ID] = client
}

// remove removes a client from the hub.
func (h *SSEHub) remove(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if client, ok := h.clients[clientID]; ok {
		close(client.Send)
		delete(h.clients, clientID)
	}
}

// broadcast sends a message to the appropriate clients.
func (h *SSEHub) broadcast(message *SSEMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.BufferSize > 0 {
		if len(h.buffer) >= h.BufferSize {
			// Remove the oldest message
			h.buffer = h.buffer[1:]
		}
		h.buffer = append(h.buffer, message)
	}

	for _, client := range h.clients {
	clientLoop:
		for _, target := range message.Targets {
			for _, channel := range client.Channels {
				if target == channel {
					client.Send <- message.Data
					break clientLoop // move to the next client
				}
			}
		}
	}
}

// autoChannels generates the channels for a user.
func autoChannels(userID, role string) []string {
	return []string{
		"all",
		"role:" + role,
		"user:" + userID,
	}
}
