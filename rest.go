package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
)

func handleRooms(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		rooms := hub.GetWaitingRooms()

		type RoomInfo struct {
			RoomID       string `json:"room_id"`
			CustomerName string `json:"customer_name"`
			Language     string `json:"language"`
		}

		var result []RoomInfo

		for _, room := range rooms {
			result = append(result, RoomInfo{
				RoomID:       room.ID,
				CustomerName: room.Customer.Name,
				Language:     room.Customer.Language,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

func handleSetProfile(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req SetProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.Name == "" || req.Language == "" {
			http.Error(w, "name and language are required", http.StatusBadRequest)
			return
		}

		agent := NewClient(req.Name, req.Language)
		hub.AddClient(agent)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SetProfileResponse{
			Token: agent.Token,
		})
	}
}

func handleStartChat(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req StartChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.Name == "" || req.Content == "" {
			http.Error(w, "name and content are required", http.StatusBadRequest)
			return
		}

		customer := NewClient(req.Name, "")
		hub.AddClient(customer)

		room := hub.CreateRoom(customer)
		msg := ChatMessage{
			Type:    "message",
			RoomID:  room.ID,
			From:    customer.Name,
			Content: req.Content,
		}
		room.Messages = append(room.Messages, msg)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(StartChatResponse{
			Token:  customer.Token,
			RoomID: room.ID,
		})
	}
}

func handleJoinRoom(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			http.Error(w, "token required", http.StatusUnauthorized)
			return
		}

		agent, ok := hub.GetClient(token)
		if !ok {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		if hub.IsAgentInRoom(agent.Token) {
			http.Error(w, "already in a room", http.StatusConflict)
			return
		}

		var req RoomRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		room, err := hub.JoinRoom(req.RoomID, agent)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Notify the customer via WebSocket if they're connected
		if room.Customer != nil && room.Customer.Connection != nil {
			notification, _ := json.Marshal(RoomJoinedResponse{
				Type:   "room_joined",
				RoomID: room.ID,
			})
			room.Customer.Connection.Write(r.Context(), websocket.MessageText, notification)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(RoomJoinedResponse{
			Type:   "room_joined",
			RoomID: room.ID,
		})
	}
}

func handleEndChat(hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			http.Error(w, "token required", http.StatusUnauthorized)
			return
		}

		client, ok := hub.GetClient(token)
		if !ok {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		var req RoomRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		room, ok := hub.GetRoom(req.RoomID)
		if !ok {
			http.Error(w, "room not found", http.StatusNotFound)
			return
		}

		// Figure out who ended it and who needs to be notified
		var reason string
		var other *Client
		if room.Customer != nil && room.Customer.Token == client.Token {
			reason = "customer_left"
			other = room.Agent
			room.Status = RoomClosed
			hub.RemoveRoom(room.ID)
		} else {
			reason = "agent_left"
			other = room.Customer
			room.Status = RoomClosing
			room.Agent = nil

			room.CloseTimer = time.AfterFunc(5*time.Minute, func() {
				room.Status = RoomClosed
				hub.RemoveRoom(room.ID)
				if room.Customer != nil && room.Customer.Connection != nil {
					notification, _ := json.Marshal(ChatEndedResponse{
						Type:   "chat_ended",
						RoomID: room.ID,
						Reason: "closed",
					})
					room.Customer.Connection.Write(context.Background(), websocket.MessageText, notification)
				}
			})
		}

		// Notify the other participant via WebSocket
		if other != nil && other.Connection != nil {
			notification, _ := json.Marshal(ChatEndedResponse{
				Type:   "chat_ended",
				RoomID: room.ID,
				Reason: reason,
			})
			other.Connection.Write(r.Context(), websocket.MessageText, notification)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ChatEndedResponse{
			Type:   "chat_ended",
			RoomID: room.ID,
			Reason: reason,
		})
	}
}