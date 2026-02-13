package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/coder/websocket"
)

func prepareMessage(translator *Translator, room *Room, sender *Client, recipient *Client, content string) ChatMessage {
	msg := ChatMessage{
		Type:    "message",
		RoomID:  room.ID,
		From:    sender.Name,
		Content: content,
	}

	// Detect customer language if unknown
	if sender == room.Customer && sender.Language == "" {
		lang, err := translator.DetectLanguage(content)
		if err != nil {
			slog.Error("failed to detect language", "error", err)
			return msg
		}
		sender.Language = strings.TrimSpace(lang)
		slog.Info("detected language", "client", sender.Name, "language", sender.Language)
	}

	// Skip if either language is unknown or they're the same
	if sender.Language == "" || recipient.Language == "" || sender.Language == recipient.Language {
		return msg
	}

	// Translate
	translated, err := translator.Translate(content, sender.Language, recipient.Language)
	if err != nil {
		slog.Error("translation failed", "error", err)
		return msg
	}
	translated = strings.TrimSpace(translated)

	// Customer sees translated only, agent sees both
	if recipient == room.Customer {
		msg.Content = translated
	} else {
		msg.TranslatedContent = translated
	}

	return msg
}

func handleWebSocket(hub *Hub, translator *Translator, limiter *RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "token required", http.StatusUnauthorized)
			return
		}

		roomID := r.URL.Query().Get("room_id")
		if roomID == "" {
			http.Error(w, "room_id required", http.StatusBadRequest)
			return
		}

		client, ok := hub.GetClient(token)
		if !ok {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		room, ok := hub.GetRoom(roomID)
		if !ok {
			http.Error(w, "room not found", http.StatusNotFound)
			return
		}

		// Verify this client belongs to this room
		if (room.Customer == nil || room.Customer.Token != token) &&
			(room.Agent == nil || room.Agent.Token != token) {
			http.Error(w, "you are not in this room", http.StatusForbidden)
			return
		}

		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			slog.Error("websocket accept error", "error", err)
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		client.Connection = conn
		defer func() { client.Connection = nil }()
		slog.Info("websocket connected", "client", client.Name, "room", room.ID)

		ctx := context.Background()

		// Find the recipient
		var recipient *Client
		if room.Customer.Token == client.Token {
			recipient = room.Agent
		} else {
			recipient = room.Customer
		}

		// Send message history to the agent on connect
		if client == room.Agent {
			for _, msg := range room.Messages {
				chatMsg := prepareMessage(translator, room, room.Customer, client, msg.Content)
				data, _ := json.Marshal(chatMsg)
				if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
					slog.Error("failed to deliver history", "client", client.Name, "error", err)
				}
			}
		}

		for {
			_, data, err := conn.Read(ctx)
			if err != nil {
				slog.Info("client disconnected", "client", client.Name, "error", err)
				break
			}

			var msg struct {
				Content string `json:"content"`
			}
			if err := json.Unmarshal(data, &msg); err != nil {
				slog.Warn("invalid json", "client", client.Name, "error", err)
				continue
			}

			slog.Info("message received", "client", client.Name, "content", msg.Content)

			// Rate limit check
			if !limiter.Allow(client.Token) {
				errMsg, _ := json.Marshal(ErrorResponse{
					Type:    "error",
					Message: "rate limit exceeded",
				})
				conn.Write(ctx, websocket.MessageText, errMsg)
				continue
			}

			// Record in history
			room.Messages = append(room.Messages, ChatMessage{
				Type:    "message",
				RoomID:  room.ID,
				From:    client.Name,
				Content: msg.Content,
			})

			// Reject messages to a closed room
			if room.Status == RoomClosed {
				errMsg, _ := json.Marshal(ErrorResponse{
					Type:    "error",
					Message: "room is closed",
				})
				conn.Write(ctx, websocket.MessageText, errMsg)
				break
			}

			// Customer sends a message while room is closing — cancel the timer, reopen the room
			if room.Status == RoomClosing && room.Customer != nil && room.Customer.Token == client.Token {
				if room.CloseTimer != nil {
					room.CloseTimer.Stop()
					room.CloseTimer = nil
				}
				room.Status = RoomWaiting
				slog.Info("room reopened by customer", "room", room.ID)
			}

			// If recipient isn't connected, skip live delivery (message is already in history)
			if recipient == nil || recipient.Connection == nil {
				slog.Info("message recorded", "room", room.ID, "reason", "recipient not connected")
				continue
			}
			chatMsg := prepareMessage(translator, room, client, recipient, msg.Content)
			data, _ = json.Marshal(chatMsg)
			if err := recipient.Connection.Write(ctx, websocket.MessageText, data); err != nil {
				slog.Error("failed to send message", "recipient", recipient.Name, "error", err)
			}
		}
	}
}
