package tinysse

import . "github.com/cdvelop/tinystring"

// SSEMessage is the structure for messages sent over SSE.
// The Targets field is for internal use only and is not sent to the client.
type SSEMessage struct {
    ID        string   `json:"id"`
    HandlerID uint8    `json:"handler_id"`
    Data      []byte   `json:"data"`
    Targets   []string `json:"-"` // Ignored in JSON, internal use ONLY
}

// SSEError represents an error in the SSE service.
type SSEError struct {
    Type    MessageType
    Err     error
    Context any
}
