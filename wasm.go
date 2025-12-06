//go:build wasm

package tinysse

import (
	"encoding/json"
	"syscall/js"

	"github.com/cdvelop/tinystring"
)

// TinySSE is the main struct for the library.
type TinySSE struct {
	config             *Config
	lastEventID        string
	es                 js.Value
	reconnectAttempts int
}

// Connect establishes a connection to the SSE endpoint.
func (s *TinySSE) Connect() {
	token, err := s.config.TokenProvider()
	if err != nil {
		if s.config.OnError != nil {
			s.config.OnError(&SSEError{Type: tinystring.Msg.Auth, Err: err, Context: "failed to get token"})
		}
		return
	}

	url := s.config.Endpoint + "?token=" + token
	if s.lastEventID != "" {
		url += "&lastEventId=" + s.lastEventID
	}
	s.es = js.Global().Get("EventSource").New(url)

	s.es.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		s.reconnectAttempts = 0 // Reset on successful message
		if s.config.OnMessage != nil {
			event := args[0]
			var msg SSEMessage
			err := json.Unmarshal([]byte(event.Get("data").String()), &msg)
			if err != nil {
				if s.config.OnError != nil {
					s.config.OnError(&SSEError{Type: tinystring.Msg.Parse, Err: err, Context: "failed to unmarshal message"})
				}
				return nil
			}
			s.lastEventID = event.Get("lastEventId").String()
			s.config.OnMessage(&msg)
		}
		return nil
	}))

	s.es.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if s.config.OnError != nil {
			err := &SSEError{Type: tinystring.Msg.Connect, Err: js.Error{Value: args[0]}, Context: "connection error"}
			s.config.OnError(err)
		}
		// Only reconnect on non-fatal errors
		if s.es.Get("readyState").Int() != 2 { // CLOSED
			s.reconnect()
		}
	}))
}

func (s *TinySSE) reconnect() {
	s.es.Call("close")
	if s.config.MaxReconnectAttempts > 0 && s.reconnectAttempts >= s.config.MaxReconnectAttempts {
		if s.config.OnError != nil {
			s.config.OnError(&SSEError{Type: tinystring.Msg.Connect, Err: js.Error{Value: js.ValueOf("max reconnect attempts reached")}, Context: "reconnection failed"})
		}
		return
	}
	delay := s.config.RetryInterval * (1 << s.reconnectAttempts)
	if delay > s.config.MaxRetryDelay {
		delay = s.config.MaxRetryDelay
	}
	js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		s.Connect()
		return nil
	}), delay)
	s.reconnectAttempts++
}
