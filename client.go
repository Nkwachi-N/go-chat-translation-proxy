package main

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/coder/websocket"
)

type Client struct {
	Token      string
	Name       string
	Language   string
	Connection *websocket.Conn
}

func NewClient(name string, language string) *Client {
	token := generateToken()
	return &Client{
		Token:    token,
		Name:     name,
		Language: language,
	}
}

func generateToken() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
