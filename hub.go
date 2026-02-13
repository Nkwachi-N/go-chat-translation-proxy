package main

import (
	"fmt"
	"log/slog"
	"sync"
)

type Hub struct {
	Clients map[string]*Client
	Rooms   map[string]*Room
	mu      sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		Clients: make(map[string]*Client),
		Rooms:   make(map[string]*Room),
	}
}

func (h *Hub) AddClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Clients[client.Token] = client
	slog.Info("client added", "token", client.Token, "total", len(h.Clients))
}

func (h *Hub) RemoveClient(token string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.Clients, token)
	slog.Info("client removed", "token", token, "total", len(h.Clients))
}

func (h *Hub) CreateRoom(customer *Client) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()
	roomID := "room_" + generateToken()
	room := NewRoom(roomID, customer)
	h.Rooms[roomID] = room
	slog.Info("room created", "room", roomID, "total", len(h.Rooms))
	return room
}

func (h *Hub) JoinRoom(roomID string, agent *Client) (*Room, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.Rooms[roomID]
	if !ok {
		return nil, fmt.Errorf("room not found: %s", roomID)
	}
	if room.Status != RoomWaiting {
		return nil, fmt.Errorf("room is not available: %s", roomID)
	}
	if room.Agent != nil {
		return nil, fmt.Errorf("agent is already in room: %s", roomID)
	}
	if room.Customer == nil {
		return nil, fmt.Errorf("customer is required: %s", roomID)
	}
	room.Agent = agent
	room.Status = RoomActive
	slog.Info("agent joined room", "room", roomID, "total", len(h.Rooms))
	return room, nil
}

func (h *Hub) GetWaitingRooms() []*Room {
	h.mu.Lock()
	defer h.mu.Unlock()
	var waitingRooms []*Room
	for _, room := range h.Rooms {
		if room.Status == RoomWaiting {
			waitingRooms = append(waitingRooms, room)
		}
	}
	return waitingRooms
}

func (h *Hub) GetRoom(roomID string) (*Room, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	room, ok := h.Rooms[roomID]
	return room, ok
}

func (h *Hub) IsAgentInRoom(agentToken string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, room := range h.Rooms {
		if room.Agent != nil && room.Agent.Token == agentToken && room.Status == RoomActive {
			return true
		}
	}
	return false
}

func (h *Hub) RemoveRoom(roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.Rooms, roomID)
	slog.Info("room removed", "room", roomID, "total", len(h.Rooms))
}

func (h *Hub) GetClient(token string) (*Client, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	client, ok := h.Clients[token]
	return client, ok
}
