# Chat Translation Proxy

A customer support chat backend with real-time translation. Customers and support reps chat in their own languages — the server translates messages on the fly using Ollama.

My tutor? **Claude Code** and the **Go Docs** (again).

## A Note on Building This

This is my second Go project (after a [todo CLI](https://github.com/nkwachi/todo-cli)). It started as a simple chat translation proxy, but questioning the design at every step turned it into something more focused: a customer support chat backend. 

Building backend feels closer to the product than frontend ever did. How the API is shaped, what a client has to do to start a chat, how errors surface. These are design decisions and they directly affect the user experience.

Even though this is a learning project, I tried to treat every decision seriously. I questioned the architecture, pushed back when something felt wrong, and kept coming back to simplicity and readability as my guiding principles.

The guiding principle throughout: **if I can't explain why something exists, it probably shouldn't.**
    
## What It Does

Customers start a chat, a support agent joins, and messages between them get auto-translated if they speak different languages. Powered by Ollama running locally.

**Architecture**: REST for actions, WebSocket for real-time chat.

```
Customer (Portuguese)          Server              Rep (English)
       │                         │                       │
       ├── POST /start-chat ────►│                       │
       │◄── {token, room_id} ────┤                       │
       │                         │◄── POST /set-profile ─┤
       │                         │◄── GET /rooms ────────┤
       │                         ├── [waiting rooms] ───►│
       │                         │◄── POST /join-room ───┤
       │                         │                       │
       ├── WS /ws?token&room_id ►│◄── WS /ws?token&room_id┤
       │                         │                       │
       ├── "Preciso de ajuda" ──►│                       │
       │                         ├── "I need help" ─────►│
       │                         │   (+ original text)   │
       │                         │◄── "What happened?" ──┤
       │◄── "O que aconteceu?" ──┤                       │
       │   (translated only)     │                       │
```

## How I'm Building It (The Journey)

### Phase 1: WebSocket Echo Server
- [x] HTTP server with `/` and `/ws` endpoints
- [x] Accept WebSocket connections
- [x] Read messages in a loop, echo them back
- [x] JSON message format with separate request/response structs
- [x] Handle invalid JSON without crashing
- [x] Test with `websocat`

### Phase 2: Rooms, Routing & Identity
- [x] Client struct (token, name, language, connection)
- [x] Room struct (room ID, customer, agent, status, pending messages, close timer)
- [x] Hub struct with mutex (manages clients + rooms)
- [x] `POST /start-chat` — customer sends name + first message, gets token + room ID
- [x] `POST /set-profile` — agent sets name + language, gets token
- [x] `GET /rooms` — list waiting rooms
- [x] `POST /join-room` — agent joins a room
- [x] `POST /end-chat` — either side ends the chat
- [x] `GET /ws?token=xxx&room_id=yyy` — WebSocket for chat messages + server notifications
- [x] Agent can only be in one room at a time
- [x] End chat: customer closes immediately, agent triggers 5-min timer
- [x] Message queuing when recipient isn't connected
- [x] WebSocket notifications (room_joined, chat_ended)
- [x] Connection cleanup on disconnect

### Phase 3: Ollama Translation
- [x] HTTP client to call Ollama's API
- [x] Auto-detect customer language from first message
- [x] `Translate()` function
- [x] Skip if same language, fallback if translation fails
- [x] Customer sees translated text only, agent sees both original + translated

### Phase 4: Test Client
- [x] `customer.html` — enter name, type first message to start chat, wait for agent, chat, end chat
- [x] `agent.html` — set name + language, list waiting rooms, join, chat, end chat
- [x] Serve static files from Go
- [x] Customer sees translated text only, agent sees both original + translated
- [x] Message history — new agents see the full conversation when joining a reopened room

### Phase 5: Caching & Rate Limiting
- [x] Translation cache with lazy TTL (10-minute expiry, checked on read)
- [x] Rate limiting per client (sliding window, 10 messages per minute)
- [x] Thread-safe cache with `sync.RWMutex`
- [x] Thread-safe rate limiter with `sync.Mutex`

### Phase 6: Polish
- [x] Structured logging with `log/slog` (key-value pairs instead of format strings)
- [x] Config via environment variables (`PORT`, `OLLAMA_URL`, `OLLAMA_MODEL`, `RATE_LIMIT`, `CACHE_TTL`)
- [x] Graceful shutdown (signal handling, `http.Server.Shutdown`)
- [x] Health check endpoint (`GET /health`)
- [x] Unit tests for Hub and RateLimiter (11 tests)

## Project Structure

```
chat-translation-proxy/
├── main.go              # Entry point, config, routes, graceful shutdown
├── config.go            # Config struct, environment variable loading
├── rest.go              # REST handlers (start-chat, set-profile, rooms, join-room, end-chat)
├── websocket.go         # WebSocket handler (auth, message routing, history, translation)
├── translate.go         # Ollama client (language detection, translation, caching)
├── message.go           # Request/response structs for REST and WebSocket
├── client.go            # Client struct, token generation
├── hub.go               # Hub struct, client/room management, mutex
├── hub_test.go          # Hub unit tests
├── room.go              # Room struct, room statuses
├── ratelimit.go         # Per-client rate limiter (sliding window)
├── ratelimit_test.go    # Rate limiter unit tests
├── static/
│   ├── customer.html    # Customer test page
│   └── agent.html       # Agent test page
├── go.mod
└── README.md
```

## Things I Learned Along The Way

### Use the Right Tool: REST vs WebSocket
The first version routed everything through WebSocket. Room creation, profile setup, chat messages, all as JSON types over one connection. Splitting actions into REST and keeping WebSocket for real-time chat only made the design much cleaner. REST is request/response. WebSocket is persistent. Use each for what it's good at.

### WebSockets Are Not REST With Extra Steps
REST: client asks, server answers, connection closes. WebSocket: client connects once, connection stays open, both sides can send at any time. The server can push to the client without being asked. That's the whole reason WebSocket exists.

### The WebSocket Upgrade
A WebSocket connection starts as a normal HTTP request. The client says "upgrade this to WebSocket" and the server says yes. After that it's no longer HTTP. In Go, `websocket.Accept(w, r, nil)` handles the handshake.

### conn.Read() Blocks
The infinite `for` loop in the WebSocket handler doesn't burn CPU. `conn.Read()` blocks, meaning the goroutine sleeps until the client sends a message or disconnects. Between messages it costs basically nothing.

### defer Is a Cleanup Stack
`defer` schedules a function call to run when the surrounding function exits. Go pushes deferred calls onto a stack and runs them in reverse order (last in, first out). Arguments are evaluated immediately when `defer` is called, not when it runs.

### := With Mixed Variables
`:=` requires at least one new variable on the left side. If some already exist, Go creates the new ones and reuses the existing ones. `response, err := json.Marshal(msg)` creates `response` but reuses `err` if it was already declared.

### continue vs break
In a network read loop, `continue` skips bad input (like invalid JSON) and goes back to reading. The connection stays alive. `break` exits the loop entirely, which you use when the connection itself is broken.

### Structs Are Value Types
Go doesn't have objects or reference types. Structs are copied when passed to functions. Use pointers (`*Room`, `*Client`) when you want shared access so everyone references the same data instead of separate copies.

### Custom Types as Enums
Go doesn't have enums. Instead you do `type RoomStatus string` with `const` values. The compiler treats it as a separate type so you can't accidentally pass a random string where a `RoomStatus` is expected.

### sync.Mutex
Multiple goroutines accessing the same map will crash. `Mutex` is a lock. `Lock()` means "only I can touch this now", `Unlock()` means "go ahead". Always pair with `defer` so it unlocks even on errors.

### nil Slices Are Safe
`len(nil)` returns 0. `range` over nil does nothing. `append` to nil works. The only thing that panics is indexing into nil (`slice[0]`).

### Goroutines Are Implicit in HTTP
Go's HTTP server runs each handler in its own goroutine automatically. You don't need to write `go handler()`. It happens behind the scenes. That's why multiple WebSocket clients can connect simultaneously.

### Channels vs Mutex
Two ways to handle concurrent access to shared data. Mutex: multiple goroutines access the data but only one at a time (lock/unlock). Channels: only one goroutine owns the data, everyone else communicates through a queue. Channels are the "Go way" but mutex is simpler for straightforward cases.

### Channels Are FIFO
Channels are first-in-first-out queues. If goroutine A sends before goroutine B, the receiver processes A first. Order is preserved.

### Maps Must Be Initialized
`make(map[string]*Client)` is required. An uninitialized (nil) map will panic on write. Reading from a nil map is fine and returns the zero value.

### Modules vs Packages vs Libraries
A module is your whole project (defined by `go.mod`). A package is a directory of Go files that can be imported. A library is a package you publish for others to use. You can have many packages in one module.

### fmt.Errorf Creates Errors
`fmt.Errorf("room %s not found", roomID)` returns an `error` type with a formatted message. It's like `fmt.Sprintf` but for errors.

### json.NewEncoder for HTTP Responses
`json.NewEncoder(w).Encode(data)` writes JSON directly to an HTTP response. Simpler than doing `json.Marshal` then `w.Write` when you're writing to a response writer.

### json.NewDecoder for Request Bodies
`json.NewDecoder(r.Body).Decode(&req)` reads JSON from an HTTP request body into a struct. It's the mirror of `json.NewEncoder(w).Encode()` for responses. Decoder reads, encoder writes.

### http.Error Is a Shorthand
`http.Error(w, "message", http.StatusBadRequest)` sets the status code and writes the error message in one call. Cleaner than manually setting headers and writing bytes.

### Functions Returning Functions
`handleStartChat(hub *Hub) http.HandlerFunc` is a function that returns a function. You call it once at startup with the hub, and it returns a handler that has `hub` baked in. This is how you pass dependencies to HTTP handlers in Go without global variables.

### time.AfterFunc Runs in a New Goroutine
`time.AfterFunc(duration, func)` schedules a function to run after a delay in its own goroutine. It returns a `*time.Timer` you can call `.Stop()` on to cancel. Useful for "do X unless Y happens first" patterns.

### Garbage Collection Keeps Pointers Alive
When you delete something from a map, only the map's reference is removed. If another variable still points to the same object, the object stays in memory. Go's garbage collector only frees memory when nothing references it anymore. No dangling pointers.

### r.Context() vs context.Background()
`r.Context()` is tied to the HTTP request lifecycle. Use it for short-lived work inside a handler. `context.Background()` has no parent, no timeout, no cancellation. Use it for long-lived operations like WebSocket connections that outlive the original request.

### Making HTTP Requests from Go
`http.Client` is the outbound equivalent of `http.HandleFunc`. `client.Post(url, contentType, body)` sends a POST request. `bytes.NewReader(data)` wraps a byte slice into a reader. Always `defer resp.Body.Close()` or you leak connections.

### map[string]any for Quick JSON
When you need to build a JSON object without defining a struct, `map[string]any` works. `any` is Go's alias for `interface{}` meaning any type. Good for one-off API calls where a named struct would be overkill.

### Private Functions Start Lowercase
In Go, a function starting with a lowercase letter is unexported (private to the package). `generate()` is private, `Translate()` is public. Same rule applies to struct fields, types, and everything else.

### History vs Queues
The first version had `PendingMessages` — a temporary queue for undelivered messages. But once I added `Messages` (a permanent record of every message in the room), the queue became redundant. The customer is always connected during the chat, so there's never a pending message for them. And the agent gets the full history on connect. One field replaced two concerns. Simpler data model, simpler code.

### http.FileServer for Static Files
`http.FileServer(http.Dir("static"))` serves files from a directory. Pair it with `http.StripPrefix("/static/", ...)` so `/static/customer.html` maps to `static/customer.html` on disk. Go handles MIME types, caching headers, and directory listings automatically.

### JavaScript WebSocket API
`new WebSocket(url)` connects. `ws.send(data)` sends. `ws.onmessage` receives. `ws.onclose` fires on disconnect. It mirrors Go's WebSocket API closely — both sides read and write messages, both sides can close. The browser handles the upgrade handshake automatically.

### fetch for REST Calls
JavaScript's `fetch(url, options)` is the equivalent of Go's `http.Client`. Set method, headers, and body. It returns a Promise, so `await fetch(...)` makes it read like synchronous code. `resp.json()` parses the response body, like `json.NewDecoder` in Go.

### Small Models Have Language Limits
llama3.2 handles European languages well (Portuguese, Spanish, French, German) but can't reliably output CJK characters (Japanese, Chinese, Korean). When testing translation with Japanese, the model returned Finnish instead. This isn't a code bug — it's a model capability limitation. The fix is using a larger model, not changing the code.

### sync.RWMutex vs sync.Mutex
`Mutex` is a single lock — one goroutine at a time, period. `RWMutex` has two modes: `RLock` for reading (multiple goroutines can hold this simultaneously) and `Lock` for writing (exclusive, waits for all readers to finish). Use `RWMutex` when reads are far more frequent than writes, like a translation cache. Use `Mutex` when every access is a write, like a rate limiter.

### Maps as Caches
A Go map with a struct value (text + timestamp) is a simple in-memory cache. The key is a composite string (`fromLang:toLang:text`), the value is the cached result. No external dependencies, no serialization. Good enough for a single-server app.

### Lazy TTL Expiration
Instead of running a background goroutine to clean up expired entries, check the timestamp on read. If the entry is too old, treat it as a cache miss. Simpler, no extra goroutine, no timer management. The tradeoff: stale entries sit in memory until someone reads them. For a small cache, that's fine.

### Sliding Window Rate Limiting
Track timestamps of each client's recent messages. On each new message, filter out timestamps older than the window, count what's left. If over the limit, reject. Old messages "fall off" naturally as time passes. Called "sliding" because the window moves with the current time instead of using fixed intervals.

### log/slog Over log
`log.Printf` formats strings. `slog.Info` uses key-value pairs: `slog.Info("client added", "token", token, "total", count)`. The output is structured, which means it can be parsed by log aggregation tools. Three levels: `slog.Info` for normal events, `slog.Warn` for recoverable issues, `slog.Error` for failures.

### os.Getenv for Config
`os.Getenv("PORT")` reads an environment variable. Returns empty string if not set. Wrap it in a helper that returns a default: `envOrDefault("PORT", "8080")`. No external config library needed. For numbers and durations, use `strconv.Atoi` and `time.ParseDuration`.

### Graceful Shutdown
`http.ListenAndServe` blocks forever and can't be stopped cleanly. Instead: create `http.Server`, run `ListenAndServe` in a goroutine, wait for `SIGINT`/`SIGTERM` via `os/signal`, then call `srv.Shutdown(ctx)`. Shutdown stops accepting new connections and waits for active ones to finish, up to the context deadline.

### Channels for Signaling
`make(chan os.Signal, 1)` creates a buffered channel. `signal.Notify(quit, syscall.SIGINT)` sends to it when Ctrl+C is pressed. `<-quit` blocks until a signal arrives. Channels aren't just for data — they're also for coordination between goroutines.

### Go Testing Basics
Test files end in `_test.go`. Test functions start with `Test` and take `*testing.T`. `t.Fatal` stops the test. `t.Errorf` logs a failure but continues. Run with `go test -v .`. Tests live next to the code they test, same package, no special setup.

### Design Before You Code
I changed the architecture three times before writing most of the code: standalone app, then translation library, then customer support chat. Each pivot came from asking "what's the actual use case?" rather than jumping into implementation. The time spent on design wasn't wasted. It prevented building the wrong thing.

## Areas for Improvement

Things I'd fix or add if I kept building on this. Left intentionally as learning markers — the next project ([url-shortener](https://github.com/nkwachi/url-shortener)) addresses most of these.

### Race Conditions on Room Fields
`Room.Status`, `Room.Agent`, and `Room.Messages` are read and written from multiple goroutines (WebSocket handlers, REST handlers, close timer) without synchronization. The hub mutex protects the maps, but not the individual room fields. Fix: either lock the hub mutex around all room field access, or add a per-room mutex.

### Health Endpoint Reads Maps Without Mutex
The `/health` handler reads `hub.Clients` and `hub.Rooms` directly to get counts, but doesn't hold the hub mutex. Another goroutine could be modifying these maps at the same time. Fix: add `hub.ClientCount()` and `hub.RoomCount()` methods that lock before reading.

### History Replay Assumes Customer as Sender
When an agent connects and receives message history, `prepareMessage` always uses `room.Customer` as the sender. Messages the previous agent sent get treated as customer messages and translated the wrong way. Fix: store the sender role in `ChatMessage` and use it during replay.

### Token in Query String
WebSocket auth uses `?token=xxx` in the URL. This means tokens show up in server access logs, browser history, and any proxy logs. Fine for a learning project, but in production you'd use a cookie or the first WebSocket message for auth.

### End Chat Missing Ownership Check
`POST /end-chat` verifies the token is valid but doesn't check if the caller actually belongs to the room they're trying to close. Any authenticated user could end any room. Fix: verify `room.Customer.Token == token || room.Agent.Token == token`.

### Client/Room Leak
`RemoveClient` exists on the hub but is never called. When a customer disconnects or a room closes, the client stays in `hub.Clients` forever. Over time, the map grows without bound. Fix: call `RemoveClient` in the appropriate cleanup paths.

### No Interfaces
Everything is concrete types. The hub, translator, and rate limiter are passed directly. This makes unit testing handlers impossible without running the real dependencies. Fix: define interfaces (`Store`, `Translator`) and pass those instead — the next project covers this.

### No HTTP Middleware
Auth checking, rate limiting, and logging are done inline in handlers. This leads to duplicated code and makes it hard to apply cross-cutting concerns consistently. Fix: extract into middleware functions that wrap handlers — the next project covers this.

## Resources

- [coder/websocket docs](https://pkg.go.dev/github.com/coder/websocket)
- [Ollama API docs](https://github.com/ollama/ollama/blob/main/docs/api.md)
- [Go by Example](https://gobyexample.com/)
- [Go Concurrency Patterns](https://go.dev/blog/pipelines)

---

Built by learning Go one phase at a time.