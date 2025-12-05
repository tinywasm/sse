//go:build wasm

package tinysse

import (
	"encoding/json"
	"syscall/js"
	"time"
)

// Client is the SSE client for WASM.
type Client struct {
	cfg         *Config
	eventSource js.Value
	lastEventID string
	url         string
	isReconnecting bool
}

// NewClient creates a new SSE client.
func NewClient(cfg *Config) *Client {
	return &Client{
		cfg: cfg,
		url: cfg.Endpoint,
	}
}

// Connect starts the SSE client.
func (c *Client) Connect() {
	go c.run()
}

// run is the main loop for the client.
func (c *Client) run() {
	for {
		c.connect()
		if !c.isReconnecting {
			break
		}
		time.Sleep(time.Duration(c.cfg.RetryInterval) * time.Second)
	}
}

// connect establishes a connection to the SSE server.
func (c *Client) connect() {
	token, err := c.cfg.TokenProvider()
	if err != nil {
		if c.cfg.OnError != nil {
			c.cfg.OnError(err)
		}
		return
	}

	url := c.url + "?token=" + token
	if c.lastEventID != "" {
		url += "&lastEventId=" + c.lastEventID
	}

	c.eventSource = js.Global().Get("EventSource").New(url)

	c.eventSource.Set("onopen", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if c.cfg.OnConnect != nil {
			c.cfg.OnConnect("") // clientID is not available in wasm
		}
		c.isReconnecting = false
		return nil
	}))

	c.eventSource.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		c.lastEventID = event.Get("lastEventId").String()
		data := event.Get("data").String()
		var msg SSEMessage
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			if c.cfg.OnError != nil {
				c.cfg.OnError(err)
			}
			return nil
		}
		if c.cfg.OnMessage != nil {
			c.cfg.OnMessage(&msg)
		}
		return nil
	}))

	c.eventSource.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// EventSource will automatically retry on network errors.
		// We only need to handle manual reconnection for auth errors.
		if c.eventSource.Get("readyState").Int() == 2 { // CLOSED
			c.isReconnecting = true
			c.eventSource.Call("close")
			if c.cfg.OnDisconnect != nil {
				c.cfg.OnDisconnect("") // clientID is not available in wasm
			}
		}
		return nil
	}))
}

// Close closes the connection to the SSE server.
func (c *Client) Close() {
	if !c.eventSource.IsUndefined() {
		c.eventSource.Call("close")
	}
	c.isReconnecting = false
}
