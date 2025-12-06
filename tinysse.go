package tinysse

// tinySSE is the internal struct holding shared configuration.
type tinySSE struct {
	config *Config
}

// New creates a new tinySSE instance with shared configuration.
func New(c *Config) *tinySSE {
	return &tinySSE{config: c}
}

// log prints to the configured logger if one is set.
func (t *tinySSE) log(args ...any) {
	if t.config.Log != nil {
		t.config.Log(args...)
	}
}
