package main

// --- REST request bodies ---

// StartChatRequest is sent by a customer to POST /start-chat.
type StartChatRequest struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// SetProfileRequest is sent by an agent to POST /set-profile.
type SetProfileRequest struct {
	Name     string `json:"name"`
	Language string `json:"language"`
}

// RoomRequest is used for POST /join-room and POST /end-chat.
type RoomRequest struct {
	RoomID string `json:"room_id"`
}

// --- REST response bodies ---

// StartChatResponse is returned from POST /start-chat.
type StartChatResponse struct {
	Token  string `json:"token"`
	RoomID string `json:"room_id"`
}

// SetProfileResponse is returned from POST /set-profile.
type SetProfileResponse struct {
	Token string `json:"token"`
}

// --- WebSocket messages ---

// ChatMessage is sent to deliver a message to the other participant.
type ChatMessage struct {
	Type              string `json:"type"`
	RoomID            string `json:"room_id"`
	From              string `json:"from"`
	Content           string `json:"content"`
	TranslatedContent string `json:"translated_content,omitempty"`
}

// RoomJoinedResponse is sent over WebSocket when an agent joins a room.
type RoomJoinedResponse struct {
	Type   string `json:"type"`
	RoomID string `json:"room_id"`
}

// ChatEndedResponse is sent over WebSocket when a chat ends.
type ChatEndedResponse struct {
	Type   string `json:"type"`
	RoomID string `json:"room_id"`
	Reason string `json:"reason"`
}

// ErrorResponse is sent when something goes wrong.
type ErrorResponse struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
