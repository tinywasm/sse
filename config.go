package tinysse

// Config holds the shared configuration for both Server and Client.
type Config struct {
	// Log is the centralized logger function.
	// If nil, logging is disabled.
	Log func(args ...any)
}
