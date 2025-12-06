//go:build wasm

package tinysse

import (
	"syscall/js"
	"testing"
	"time"
)

// This test requires `wasmbrowsertest` or a similar environment.
// If running in standard `go test`, it will be skipped by build tag.

func TestClientConnect(t *testing.T) {
	// We cannot easily spin up a real server in WASM environment.
	// Tests here typically verify JS interop or logic that doesn't require network
	// OR use a mock EventSource if we can inject it.
	// Since we use global `EventSource`, we can mock it in JS global scope!

	// Mock EventSource
	js.Global().Set("EventSource", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return map[string]interface{}{
			"New": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				// Return a mock EventSource instance
				es := make(map[string]interface{})
				es["Set"] = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					// Store handlers
					name := args[0].String()
					fn := args[1]
					js.Global().Set("_mock_"+name, fn)
					return nil
				})
				es["Call"] = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					return nil
				})
				// Mock readyState property
				es["readyState"] = 0
				return es
			}),
		}
	}))

	cfg := &Config{Log: testLog(t)}
	tSSE := New(cfg)
	client := tSSE.Client(&ClientConfig{
		Endpoint: "/events",
	})

	client.Connect()

	// Verify Connect called New
	// Hard to verify without more elaborate mocking, but if no panic, it worked basic JS call.
}

func TestClientOnMessage(t *testing.T) {
	// Setup mock
	var onMessage js.Value

	// Mock EventSource to capture onmessage
	js.Global().Set("EventSource", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return map[string]interface{}{
			"New": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				es := make(map[string]interface{})
				es["Set"] = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					name := args[0].String()
					if name == "onmessage" {
						onMessage = args[1]
					}
					return nil
				})
				es["Call"] = js.FuncOf(func(this js.Value, args []js.Value) interface{} { return nil })
				return es
			}),
		}
	}))

	tSSE := New(&Config{})
	client := tSSE.Client(&ClientConfig{Endpoint: "/test"})

	var received *SSEMessage
	client.OnMessage(func(msg *SSEMessage) {
		received = msg
	})

	client.Connect()

	if onMessage.IsUndefined() {
		t.Fatal("onmessage handler not set")
	}

	// Simulate incoming message
	// Event object mock
	event := map[string]interface{}{
		"data":        "hello world",
		"lastEventId": "123",
		"type":        "test-event",
	}

	// JS ValueOf map doesn't work directly for properties access via Get() on struct like objects?
	// We need to make sure `args[0].Get("data")` works.
	// js.ValueOf(map) returns a JS object where keys are properties.

	onMessage.Invoke(js.ValueOf(event))

	if received == nil {
		t.Fatal("handler not called")
	}

	verifyMessage(t, received, "test-event", []byte("hello world"))
	if received.ID != "123" {
		t.Errorf("expected ID '123', got %s", received.ID)
	}
}
