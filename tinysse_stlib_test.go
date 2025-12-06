//go:build !wasm

package tinysse

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServer_Broadcast(t *testing.T) {
	connected := make(chan struct{})
	sse := New(&Config{
		BufferSize: 10,
		TokenValidator: func(token string) (string, string, error) {
			return "testuser", "testrole", nil
		},
		OnConnect: func(clientID string) {
			close(connected)
		},
	})

	server := httptest.NewServer(sse)
	defer server.Close()

	// Create a client with a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL, nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	// Wait for the client to connect
	<-connected

	// Broadcast a message
	sse.Broadcast([]byte("hello"), []string{"all"}, 0)

	// Read the response
	scanner := bufio.NewScanner(resp.Body)
	var id, data string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "id:") {
			id = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
		}
		if strings.HasPrefix(line, "data:") {
			data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			break
		}
	}
	if id == "" {
		t.Error("did not receive message id")
	}
	if data != "hello" {
		t.Errorf("expected message 'hello', got '%s'", data)
	}
}

func TestHub(t *testing.T) {
	hub := NewHub(&Config{
		MessageBufferSize: 2,
	})
	client1 := &SSEClient{ID: "1", Channels: []string{"all", "user:1"}, Send: make(chan SSEMessage, 1)}
	client2 := &SSEClient{ID: "2", Channels: []string{"all", "user:2"}, Send: make(chan SSEMessage, 1)}
	hub.register(client1)
	hub.register(client2)

	// Test broadcast to all
	hub.Broadcast([]byte("hello all"), []string{"all"}, 0)
	<-client1.Send
	<-client2.Send

	// Test broadcast to one
	hub.Broadcast([]byte("hello user1"), []string{"user:1"}, 0)
	<-client1.Send
	select {
	case <-client2.Send:
		t.Error("client2 should not have received message")
	default:
	}

	// Test unregister
	hub.unregister(client1)
	if _, ok := hub.clients["1"]; ok {
		t.Error("client1 should be unregistered")
	}

	// Test buffer trimming
	hub.Broadcast([]byte("msg1"), []string{"all"}, 0)
	hub.Broadcast([]byte("msg2"), []string{"all"}, 0)
	hub.Broadcast([]byte("msg3"), []string{"all"}, 0)
	if len(hub.messageBuffer) != 2 {
		t.Errorf("message buffer should have 2 messages, got %d", len(hub.messageBuffer))
	}
	if string(hub.messageBuffer[0].Data) != "msg2" {
		t.Errorf("expected first message to be 'msg2', got '%s'", string(hub.messageBuffer[0].Data))
	}
}
