# Chat Translation Proxy

A customer support chat backend with real-time translation. Customers and support reps chat in their own languages — the server translates messages on the fly using Ollama.

**Tech Stack**: Go, WebSockets (github.com/coder/websocket), Ollama (llama3.2)

**Architecture**: REST for actions, WebSocket for real-time chat messages + server notifications.

```
Customer (Portuguese)          Server              Rep (English)
       │                         │                       │
       ├── POST /start-chat ────►│                       │
       │◄── {token, room_id} ────┤                       │
       │                         │                       │
       │                         │◄── POST /set-profile ─┤
       │                         │◄── GET /rooms ────────┤
       │                         ├── [waiting rooms] ───►│
       │                         │◄── POST /join-room ───┤
       │                         │                       │
       ├── WS /ws?token&room_id ►│◄── WS /ws?token&room_id┤
       │◄── "room_joined" ───────┤──── "room_joined" ──►│
       │                         ├── (pending msgs) ───►│
       │                         │                       │
       ├── "Preciso de ajuda" ──►│                       │
       │                         ├── "I need help" ─────►│
       │                         │   (+ original text)   │
       │                         │◄── "What happened?" ──┤
       │◄── "O que aconteceu?" ──┤                       │
       │   (translated only)     │                       │
       │                         │                       │
       │                         │◄── POST /end-chat ────┤
       │◄── "chat_ended" ────────┤   (5 min timer starts)│
       │                         │                       │
       │   (customer can send    │                       │
       │    a message to reopen) │                       │
```

---

## Phase 1: WebSocket Echo Server

Get a WebSocket server running. Clients connect and get their messages echoed back.

- [x] HTTP server with a `/ws` endpoint
- [x] Accept WebSocket connections
- [x] Read messages in a loop, echo them back
- [x] JSON message format
- [x] Test with `websocat`

**You'll learn**: `net/http`, WebSocket upgrade, `github.com/coder/websocket`, JSON over WebSocket, goroutines

---

## Phase 2: Rooms, Routing & Identity ✅

REST endpoints handle setup actions. WebSocket handles real-time chat messages and server notifications.

### REST Endpoints
- [x] `POST /start-chat` — customer sends name + first message, gets back token + room ID
- [x] `POST /set-profile` — agent sends name + language, gets back token
- [x] `GET /rooms` — returns list of waiting rooms (for agent page)
- [x] `POST /join-room` — agent joins a room (token required)
- [x] `POST /end-chat` — either side ends the chat (token required)

### WebSocket
- [x] `GET /ws?token=xxx&room_id=yyy` — connect with token + room ID, server links connection to client
- [x] Chat messages sent/received over WebSocket only
- [x] Server pushes notifications: `room_joined`, `chat_ended`
- [x] Pending message delivery on connect (including first message)
- [x] Message queuing when recipient isn't connected
- [x] Closed room rejection

### Identity
- [x] Server generates a token via REST (`/start-chat` or `/set-profile`)
- [x] WebSocket connection authenticated via token query parameter
- [x] Reject unknown tokens
- [x] Verify client belongs to the requested room

### Rooms
- [x] Client struct (token, name, language, connection)
- [x] Room struct (room ID, customer, agent, status, pending messages, close timer)
- [x] Hub struct with mutex (manages clients + rooms)
- [x] Room statuses: `waiting` → `active` → `closing` → `closed`
- [x] Agent can only be in one room at a time
- [x] Rooms removed from hub on close

### End Chat
- [x] Either side can call `POST /end-chat`
- [x] If customer ends it → room closes immediately, removed from hub
- [x] If agent ends it → room status = `closing`, 5-min timer starts
  - Customer sends a message within 5 min → timer cancelled, room goes back to `waiting`
  - 5 min passes with no message → room status = `closed`, customer notified, room removed

### Connection Cleanup
- [x] `client.Connection = nil` on WebSocket disconnect (via defer)

**Learned**: REST APIs in Go, JSON request/response, query parameters, `sync.Mutex`, `time.AfterFunc`, `http.Error`, functions returning functions, `r.Context()` vs `context.Background()`, garbage collection and pointer lifetimes

### REST API

```
POST /start-chat
  Request:  {"name": "John", "content": "Preciso de ajuda"}
  Response: {"token": "abc123", "room_id": "room_xyz"}

POST /set-profile
  Request:  {"name": "Sarah", "language": "en"}
  Response: {"token": "def456"}

GET /rooms
  Response: [{"room_id": "room_xyz", "customer_name": "John", "language": ""}]

POST /join-room
  Header:   Authorization: Bearer <token>
  Request:  {"room_id": "room_xyz"}
  Response: {"type": "room_joined", "room_id": "room_xyz"}

POST /end-chat
  Header:   Authorization: Bearer <token>
  Request:  {"room_id": "room_xyz"}
  Response: {"type": "chat_ended", "room_id": "room_xyz", "reason": "customer_left"}
```

### WebSocket Messages (real-time only)

```json
// Client sends:
{"content": "Olá"}

// Server delivers to recipient:
{"type": "message", "room_id": "room_xyz", "from": "John", "content": "Olá", "translated_content": "Hello"}

// Server notifications:
{"type": "room_joined", "room_id": "room_xyz"}
{"type": "chat_ended", "room_id": "room_xyz", "reason": "agent_left"}
{"type": "chat_ended", "room_id": "room_xyz", "reason": "closed"}
{"type": "error", "message": "room is closed"}
```

---

## Phase 3: Ollama Translation ✅

Translate messages between participants when their languages differ.

- [x] HTTP client to call Ollama's `/api/generate` endpoint
- [x] Auto-detect customer language from their first message
- [x] `Translate(content, fromLang, toLang)` function
- [x] Wire into message routing — translate before delivering (both pending and live messages)
- [x] Skip translation if both participants speak the same language
- [x] If translation fails, send the original message anyway
- [x] Customer sees translated text only, agent sees both original + translated
- [x] Private `generate()` helper to avoid duplicating the Ollama API call
- [x] `prepareMessage()` helper for translation + recipient-aware message formatting

**Learned**: `http.Client`, `bytes.NewReader`, `resp.Body.Close()`, `map[string]any`, private vs public functions, `strings.TrimSpace`

---

## Phase 4: Test Client ✅

Two HTML pages to test the full flow end to end.

- [x] `customer.html` — enter name, type first message to start chat, wait for agent, chat, "End Chat" button
- [x] `agent.html` — set name + language, list waiting rooms, "Join" button, chat, "End Chat" button, back to list
- [x] Serve static files from Go
- [x] Customer sees translated text only, agent sees both original + translated
- [x] Message history — new agents see full conversation when joining a reopened room
- [x] Removed `PendingMessages` in favor of `Messages` (permanent history replaces temporary queue)

**Learned**: JavaScript WebSocket API, `fetch` for REST, `http.FileServer`, `http.StripPrefix`, small LLM model limitations with CJK languages

---

## Phase 5: Caching & Rate Limiting ✅

Performance and abuse prevention.

- [x] Cache translations (same text + language pair = skip LLM)
- [x] Cache TTL (lazy expiration, 10-minute window)
- [x] Rate limit per client (sliding window, 10 messages per minute)
- [x] Thread-safe cache with `sync.RWMutex`
- [x] Thread-safe rate limiter with `sync.Mutex`

**Learned**: `sync.RWMutex` (RLock vs Lock), maps as caches, lazy TTL expiration, sliding window rate limiting, `time.Since()`, `time.Now().Add(-duration)` for cutoff times

---

## Phase 6: Polish ✅

Production touches.

- [x] Structured logging (`log/slog`) — all log.Printf replaced with slog.Info/Error/Warn + key-value pairs
- [x] Config via environment variables — `config.go` with `LoadConfig()`, `envOrDefault()` helper
- [x] Graceful shutdown — `http.Server` + signal handling (`SIGINT`/`SIGTERM`) + `srv.Shutdown(ctx)`
- [x] Health check endpoint (`GET /health`) — returns JSON with status, client count, room count
- [x] Unit tests — 7 Hub tests (`hub_test.go`), 4 RateLimiter tests (`ratelimit_test.go`), all passing

**Learned**: `log/slog` structured logging, `os.Getenv` for config, `os/signal` + channels for graceful shutdown, `testing` package (`t.Fatal`, `t.Errorf`), `http.Server.Shutdown`, `errors.Is` for error comparison

---

## Getting Started

```bash
# Install WebSocket library
go get github.com/coder/websocket

# Make sure Ollama is running
ollama serve

# Pull the model
ollama pull llama3.2

# Install websocat for testing (before the HTML client is built)
brew install websocat

# Run the server
go run .

# Test REST endpoints
curl -X POST http://localhost:8080/start-chat -d '{"name":"John","content":"Hello"}'
curl http://localhost:8080/rooms

# Test WebSocket
websocat "ws://localhost:8080/ws?token=TOKEN_FROM_START_CHAT&room_id=ROOM_ID_FROM_START_CHAT"
```

---

## Resources

- [coder/websocket docs](https://pkg.go.dev/github.com/coder/websocket)
- [Ollama API docs](https://github.com/ollama/ollama/blob/main/docs/api.md)
- [Go by Example](https://gobyexample.com/)
- [Go Concurrency Patterns](https://go.dev/blog/pipelines)