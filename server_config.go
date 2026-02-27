//go:build !wasm

package sse

// ServerConfig holds configuration strictly for the Server HTTP Handler.
type ServerConfig struct {
	// ClientChannelBuffer prevents blocking on slow clients.
	// Recommended: 10-100.
	ClientChannelBuffer int

	// HistoryReplayBuffer manages the "Last-Event-ID" replay history.
	// Recommended: Depends on message frequency.
	HistoryReplayBuffer int

	// ReplayAllOnConnect replays the full history buffer to every new client
	// on first connect (when no Last-Event-ID is provided).
	// Useful for log viewers where clients may connect after events are published.
	ReplayAllOnConnect bool

	// ChannelProvider resolves channels for each SSE connection.
	// If nil, a default provider is used that rejects all connections
	// with error "channel provider not configured".
	ChannelProvider ChannelProvider
}
