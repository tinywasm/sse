package tinysse

// Config holds the configuration for TinySSE.
type Config struct {
	BufferSize           int
	MessageBufferSize    int // The number of messages to buffer in memory.
	Endpoint             string
	RetryInterval        int
	MaxRetryDelay        int
	MaxReconnectAttempts int
	AllowedOrigins       []string

	// Callbacks
	OnConnect    func(clientID string)
	OnDisconnect func(clientID string)
	OnMessage    func(msg *SSEMessage)
	OnError      func(err error)

	// Auth
	TokenValidator func(token string) (userID, role string, err error)
	TokenProvider  func() (token string, err error)
}
