//go:build !wasm

package tinysse

// ServerConfig holds configuration strictly for the Server HTTP Handler.
type ServerConfig struct {
	// ClientChannelBuffer prevents blocking on slow clients.
	// Recommended: 10-100.
	ClientChannelBuffer int

	// HistoryReplayBuffer manages the "Last-Event-ID" replay history.
	// Recommended: Depends on message frequency.
	HistoryReplayBuffer int

	// ChannelProvider resolves channels for each SSE connection.
	// If nil, a default provider is used that rejects all connections
	// with error "channel provider not configured".
	ChannelProvider ChannelProvider
}
