package dto

type WebSocketMessage struct {
	Type    string      `json:"type"`    // Type of message (e.g., "move", "state", "broadcast", "error")
	Status  string      `json:"status"`  // Status of the message (e.g., "success", "error")
	Message string      `json:"message"` // Message content
	Payload interface{} `json:"payload"` // Message payload (can be any structured data)
}
