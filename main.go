package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := LoadConfig()
	hub := NewHub()
	translator := NewTranslator(cfg.OllamaURL, cfg.OllamaModel, cfg.CacheTTL)
	limiter := NewRateLimiter(cfg.RateLimit, cfg.RateLimitWindow)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "Server is running")
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"clients": len(hub.Clients),
			"rooms":   len(hub.Rooms),
		})
	})

	http.HandleFunc("/start-chat", handleStartChat(hub))
	http.HandleFunc("/set-profile", handleSetProfile(hub))
	http.HandleFunc("/rooms", handleRooms(hub))
	http.HandleFunc("/join-room", handleJoinRoom(hub))
	http.HandleFunc("/end-chat", handleEndChat(hub))
	http.HandleFunc("/ws", handleWebSocket(hub, translator, limiter))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	srv := &http.Server{Addr: cfg.Port}

	// Start server in a goroutine
	go func() {
		slog.Info("server started", "port", cfg.Port, "model", cfg.OllamaModel)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("server stopped")
}
