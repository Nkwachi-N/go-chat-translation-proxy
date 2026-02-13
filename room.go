package main

import "time"

type RoomStatus string

const (
	RoomWaiting RoomStatus = "waiting"
	RoomActive  RoomStatus = "active"
	RoomClosed  RoomStatus = "closed"
	RoomClosing RoomStatus = "closing"
)

type Room struct {
	ID              string
	Customer        *Client
	Agent           *Client
	Status          RoomStatus
	Messages   []ChatMessage
	CloseTimer      *time.Timer
}

func NewRoom(id string, customer *Client) *Room {
	return &Room{
		ID:       id,
		Customer: customer,
		Status:   RoomWaiting,
	}
}
