package dto

type WebSocketMessage struct {
	Type    string      `json:"type"`    // Type of message (e.g., "move", "state", "broadcast", "error")
	Payload interface{} `json:"payload"` // Message payload (can be any structured data)
}
