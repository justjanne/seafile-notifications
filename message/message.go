package message

import "encoding/json"

// Message is the message communicated between clients and server.
type Message struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
}
