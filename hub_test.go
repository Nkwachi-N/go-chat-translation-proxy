package main

import "testing"

func TestAddAndGetClient(t *testing.T) {
	hub := NewHub()
	client := NewClient("Alice", "en")
	hub.AddClient(client)

	got, ok := hub.GetClient(client.Token)
	if !ok {
		t.Fatal("expected client to be found")
	}
	if got.Name != "Alice" {
		t.Errorf("expected name Alice, got %s", got.Name)
	}
}

func TestCreateAndGetRoom(t *testing.T) {
	hub := NewHub()
	customer := NewClient("Alice", "")
	hub.AddClient(customer)

	room := hub.CreateRoom(customer)

	got, ok := hub.GetRoom(room.ID)
	if !ok {
		t.Fatal("expected room to be found")
	}
	if got.Status != RoomWaiting {
		t.Errorf("expected status waiting, got %s", got.Status)
	}
	if got.Customer.Name != "Alice" {
		t.Errorf("expected customer Alice, got %s", got.Customer.Name)
	}
}

func TestJoinRoom(t *testing.T) {
	hub := NewHub()
	customer := NewClient("Alice", "")
	agent := NewClient("Bob", "en")
	hub.AddClient(customer)
	hub.AddClient(agent)

	room := hub.CreateRoom(customer)
	joined, err := hub.JoinRoom(room.ID, agent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if joined.Status != RoomActive {
		t.Errorf("expected status active, got %s", joined.Status)
	}
	if joined.Agent.Name != "Bob" {
		t.Errorf("expected agent Bob, got %s", joined.Agent.Name)
	}
}

func TestJoinRoomAlreadyActive(t *testing.T) {
	hub := NewHub()
	customer := NewClient("Alice", "")
	agent1 := NewClient("Bob", "en")
	agent2 := NewClient("Carol", "en")

	room := hub.CreateRoom(customer)
	hub.JoinRoom(room.ID, agent1)

	_, err := hub.JoinRoom(room.ID, agent2)
	if err == nil {
		t.Fatal("expected error joining active room")
	}
}

func TestGetWaitingRooms(t *testing.T) {
	hub := NewHub()
	c1 := NewClient("Alice", "")
	c2 := NewClient("Bob", "")
	agent := NewClient("Carol", "en")

	room1 := hub.CreateRoom(c1)
	hub.CreateRoom(c2)

	hub.JoinRoom(room1.ID, agent)

	waiting := hub.GetWaitingRooms()
	if len(waiting) != 1 {
		t.Errorf("expected 1 waiting room, got %d", len(waiting))
	}
}

func TestRemoveRoom(t *testing.T) {
	hub := NewHub()
	customer := NewClient("Alice", "")
	room := hub.CreateRoom(customer)

	hub.RemoveRoom(room.ID)

	_, ok := hub.GetRoom(room.ID)
	if ok {
		t.Fatal("expected room to be removed")
	}
}

func TestIsAgentInRoom(t *testing.T) {
	hub := NewHub()
	customer := NewClient("Alice", "")
	agent := NewClient("Bob", "en")

	room := hub.CreateRoom(customer)
	hub.JoinRoom(room.ID, agent)

	if !hub.IsAgentInRoom(agent.Token) {
		t.Fatal("expected agent to be in room")
	}
	if hub.IsAgentInRoom("fake-token") {
		t.Fatal("expected fake token to not be in room")
	}
}
