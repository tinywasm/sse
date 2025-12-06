package tinysse

import (
	"github.com/cdvelop/tinystring"
)

// SSEMessage represents a message sent over SSE.
type SSEMessage struct {
	ID        string   `json:"id"`
	HandlerID uint8    `json:"handler_id"`
	Data      []byte   `json:"data"`
	Targets   []string `json:"-"` // Ignored in JSON, internal use ONLY
}

// SSEError represents an error in the SSE library.
type SSEError struct {
	Type    tinystring.MessageType
	Err     error
	Context any
}

func (e *SSEError) Error() string {
	return e.Err.Error()
}
