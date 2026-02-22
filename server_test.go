//go:build !wasm

package sse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	. "github.com/tinywasm/fmt"
)

// mockChannelProvider implements ChannelProvider for testing
type mockChannelProvider struct {
	channels []string
	err      error
}

func (m *mockChannelProvider) ResolveChannels(r *http.Request) ([]string, error) {
	return m.channels, m.err
}

func TestServerFlow(t *testing.T) {
	// 1. Setup
	cfg := &Config{Log: testLog(t)}
	tSSE := New(cfg)

	provider := &mockChannelProvider{
		channels: []string{"test-channel"},
	}

	serverCfg := &ServerConfig{
		ClientChannelBuffer: 10,
		HistoryReplayBuffer: 10,
		ChannelProvider:     provider,
	}

	server := tSSE.Server(serverCfg)

	// 2. Start Server
	ts := httptest.NewServer(server)
	defer ts.Close()

	// 3. Connect Client (Simulated HTTP request)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", ts.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	t.Log("Client connected, starting publisher...")

	// 4. Publish Message
	msgData := []byte("hello world")
	go func() {
		// Wait for connection registration
		time.Sleep(200 * time.Millisecond)
		t.Log("Publishing message...")
		server.PublishEvent("greeting", msgData, "test-channel")
		t.Log("Message published")
	}()

	// 5. Read Stream
	buf := make([]byte, 1024)
	t.Log("Reading stream...")
	n, err := resp.Body.Read(buf)
	t.Logf("Read returned: n=%d err=%v", n, err)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	output := string(buf[:n])
	t.Logf("Received: %s", output)

	// Verify output format
	if !Contains(output, "event: greeting") {
		t.Error("missing event type")
	}
	if !Contains(output, "data: hello world") {
		t.Error("missing data")
	}
	if !Contains(output, "id: ") {
		t.Error("missing id")
	}
}

func TestServerHistoryReplay(t *testing.T) {
	cfg := &Config{Log: testLog(t)}
	tSSE := New(cfg)

	provider := &mockChannelProvider{channels: []string{"all"}}
	server := tSSE.Server(&ServerConfig{
		ClientChannelBuffer: 10,
		HistoryReplayBuffer: 5,
		ChannelProvider:     provider,
	})

	// Publish some messages before connection
	server.Publish([]byte("msg1"), "all")
	server.Publish([]byte("msg2"), "all")
	server.Publish([]byte("msg3"), "all")

	// Wait a bit for processing
	time.Sleep(10 * time.Millisecond)

	// Connect with Last-Event-ID = 1 (should receive 2 and 3)
	// We assume IDs are sequential 1, 2, 3...

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Last-Event-ID", "1")

	w := httptest.NewRecorder()

	// Create a context that we can cancel to stop the server loop
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)

	// Run handler in goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		server.ServeHTTP(w, req)
	}()

	// Allow some time for replay
	time.Sleep(50 * time.Millisecond)
	cancel()  // Stop server handler
	wg.Wait() // Wait for handler to completely exit before reading body

	output := w.Body.String()

	// Should contain msg2 and msg3, but NOT msg1
	if Contains(output, "data: msg1") {
		t.Error("should not receive msg1")
	}
	if !Contains(output, "data: msg2") {
		t.Error("missing msg2")
	}
	if !Contains(output, "data: msg3") {
		t.Error("missing msg3")
	}
}

func TestDefaultChannelProvider(t *testing.T) {
	cfg := &Config{}
	tSSE := New(cfg)
	// No provider set
	server := tSSE.Server(&ServerConfig{})

	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
	if !Contains(w.Body.String(), "channel provider not configured") {
		t.Errorf("expected error message, got %s", w.Body.String())
	}
}

func TestChannelProviderError(t *testing.T) {
	cfg := &Config{}
	tSSE := New(cfg)

	provider := &mockChannelProvider{
		err: Err("auth failed"),
	}
	server := tSSE.Server(&ServerConfig{ChannelProvider: provider})

	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}
