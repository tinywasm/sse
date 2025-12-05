package tinysse

// Config holds the configuration for the TinySSE service.
type Config struct {
	BufferSize    int
	Endpoint      string
	RetryInterval int
	MaxRetryDelay int

	// OnConnect is a callback function that is executed when a client connects.
	OnConnect func(clientID string)
	// OnDisconnect is a callback function that is executed when a client disconnects.
	OnDisconnect func(clientID string)
	// OnMessage is a callback function that is executed when a message is received from the client.
	OnMessage func(msg *SSEMessage)
	// OnError is a callback function that is executed when an error occurs.
	OnError func(err error)

	// TokenValidator is a function that validates a token and returns the user ID, role, and an error.
	TokenValidator func(token string) (userID, role string, err error) // Server
	// TokenProvider is a function that provides a token.
	TokenProvider func() (token string, err error) // Client
	// AllowedOrigins is a list of allowed origins for CORS.
	AllowedOrigins []string // Server
}
