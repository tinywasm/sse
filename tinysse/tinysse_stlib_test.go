//go:build !wasm

package tinysse

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSSEHub(t *testing.T) {
	hub := newHub(10)
	client1 := &SSEClient{ID: "1", UserID: "user1", Role: "admin", Channels: autoChannels("user1", "admin"), Send: make(chan []byte, 1)}
	client2 := &SSEClient{ID: "2", UserID: "user2", Role: "user", Channels: autoChannels("user2", "user"), Send: make(chan []byte, 1)}

	hub.add(client1)
	hub.add(client2)

	if len(hub.clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(hub.clients))
	}

	// Test broadcast to all
	msg := &SSEMessage{Data: []byte("hello"), Targets: []string{"all"}}
	hub.broadcast(msg)

	select {
	case <-client1.Send:
	case <-time.After(1 * time.Second):
		t.Fatal("client1 did not receive message")
	}
	select {
	case <-client2.Send:
	case <-time.After(1 * time.Second):
		t.Fatal("client2 did not receive message")
	}

	// Test broadcast to role
	msg = &SSEMessage{Data: []byte("hello admin"), Targets: []string{"role:admin"}}
	hub.broadcast(msg)
	select {
	case <-client1.Send:
	case <-time.After(1 * time.Second):
		t.Fatal("client1 did not receive message")
	}
	select {
	case <-client2.Send:
		t.Fatal("client2 should not have received message")
	default:
	}

	hub.remove("1")
	if len(hub.clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(hub.clients))
	}
}

func TestSSEServer(t *testing.T) {
	connected := make(chan struct{})
	cfg := &Config{
		BufferSize: 10,
		TokenValidator: func(token string) (userID, role string, err error) {
			if token == "good" {
				return "user1", "admin", nil
			}
			return "", "", http.ErrNoCookie
		},
		OnConnect: func(clientID string) {
			connected <- struct{}{}
		},
	}
	server := NewServer(cfg)

	req := httptest.NewRequest("GET", "/events?token=good", nil)
	w := httptest.NewRecorder()

	go server.ServeHTTP(w, req)

	select {
	case <-connected:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for client to connect")
	}

	if len(server.hub.clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(server.hub.clients))
	}

	// Test bad token
	req = httptest.NewRequest("GET", "/events?token=bad", nil)
	w = httptest.NewRecorder()
	server.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}
