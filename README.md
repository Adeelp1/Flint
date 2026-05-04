# Flint

A high-performance HTTP/1.1 server built from raw TCP sockets in Go — no frameworks, no `net/http`. Every layer from socket to middleware is implemented from scratch to understand what production HTTP servers actually do.

---

## What this is

Most Go developers use `net/http` without knowing what it hides. Flint answers the question: *what is actually happening beneath the abstraction?*

Built over six weeks, Flint implements the full HTTP/1.1 request lifecycle — TCP listener, request parser, Trie-based router, middleware chain, response writer, worker pool, Keep-Alive, TLS, and graceful shutdown — without importing a single HTTP framework.

---

## Architecture

```
Client
  │
  │  TCP / TLS
  ▼
┌─────────────────────────────────────────────────┐
│  TCP Listener          (server.go)              │
│  net.Listen → Accept loop → connChan            │
└───────────────────┬─────────────────────────────┘
                    │  net.Conn
                    ▼
┌─────────────────────────────────────────────────┐
│  Worker Pool           (server.go)              │
│  100 goroutines reading from buffered channel   │
└───────────────────┬─────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────────┐
│  Connection Handler    (conn.go)                │
│  Keep-Alive loop · SetDeadline · bufio.Reader   │
└───────────────────┬─────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────────┐
│  Request Parser        (request.go)             │
│  Request line · Headers map · Body via          │
│  Content-Length · RemoteAddr                    │
└───────────────────┬─────────────────────────────┘
                    │  *Request
                    ▼
┌─────────────────────────────────────────────────┐
│  Middleware Chain      (middleware.go)          │
│  Logger → Auth → RateLimit → Handler            │
└───────────────────┬─────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────────────┐
│  Trie Router           (router.go)              │
│  Static + wildcard segment matching             │
│  Method dispatch · 404 / 405 handling           │
└───────────────────┬─────────────────────────────┘
                    │  *Response
                    ▼
┌─────────────────────────────────────────────────┐
│  Response Writer       (response.go)            │
│  Status line · Headers · Body · conn.Write      │
└─────────────────────────────────────────────────┘
```

Each layer has exactly one responsibility. A change to the router never touches the parser. A change to middleware never touches the transport layer. This boundary discipline is what makes the codebase navigable at scale.

---

## Features

- **Raw TCP** — `net.Listen`, `Accept` loop, `net.Conn` read/write with no HTTP library
- **HTTP/1.1 parser** — request line, headers map, body via `Content-Length`, edge case handling
- **Trie router** — O(k) path matching where k is path depth, dynamic params (`:id`), method dispatch, 404 / 405
- **Middleware chain** — composable `func(HandlerFunc) HandlerFunc` pattern, `Chain()` helper
- **Logger middleware** — method, path, status code, duration per request
- **Auth middleware** — Bearer token validation, configurable secret, 401 on failure
- **Rate limiter** — token bucket algorithm, per-IP tracking, mutex-protected, 429 on exhaustion
- **Worker pool** — 100 fixed goroutines, 1000-connection buffer channel, bounded concurrency
- **Keep-Alive** — TCP connection reuse across multiple requests, deadline reset per cycle
- **Connection timeouts** — `SetDeadline` per request, silent EOF and timeout handling
- **TLS / HTTPS** — `tls.Listen` with X.509 certificate, transport-level encryption
- **Graceful shutdown** — `sync.WaitGroup` per connection, drains in-flight requests on SIGTERM

---

## Project structure

```
flint/
├── main.go                 ← composition root — wires routes, starts server
├── handler/
│   ├── ping.go             ← GET /ping
│   ├── echo.go             ← POST /echo
│   └── home.go             ← GET /users/:id
└── server/
    ├── server.go           ← Config, Server, worker pool, TLS listener, shutdown
    ├── conn.go             ← per-connection handler, Keep-Alive loop, deadlines
    ├── request.go          ← HTTP parser, Request struct
    ├── response.go         ← Response struct, status codes, write
    ├── router.go           ← Trie data structure, dispatch, 404/405
    └── middleware.go       ← Logger, Auth, RateLimit, Chain, TokenBucket
```

---

## Getting started

**Generate a self-signed TLS certificate:**
```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes -subj "/CN=localhost"
```

**Run the server:**
```bash
go run main.go
# Flint listening on :8443
```

**Test the endpoints:**
```bash
# health check — no auth required
curl -k https://localhost:8443/ping

# protected route — requires Bearer token
curl -k -H "Authorization: Bearer 123456789abcdef" https://localhost:8443/users/42

# echo — returns request body
curl -k -X POST -d "hello flint" -H "Authorization: Bearer 123456789abcdef" https://localhost:8443/echo

# 404
curl -k https://localhost:8443/nonexistent

# 405
curl -k -X POST https://localhost:8443/ping

# 401
curl -k https://localhost:8443/users/42
```

---

## Benchmark results

All benchmarks run on a local machine (Windows, AMD Ryzen 5, 8GB RAM) using [`hey`](https://github.com/rakyll/hey): `hey -n 10000 -c 100`.

### Flint vs Go stdlib net/http

| Server | Req/sec | p50 | p95 | p99 |
|---|---|---|---|---|
| Flint (naked goroutines) | 2,985 | 31ms | 49ms | 102ms |
| Flint (worker pool) | 1,975 | 42ms | 114ms | 168ms |
| Go `net/http` stdlib | ~45,000 | 2ms | 4ms | 8ms |

**Why Flint is slower than stdlib:** Go's `net/http` has 15+ years of performance tuning. The specific gaps are:
1. **Buffer allocation** — stdlib reuses `bufio.Reader` buffers across the connection pool via `sync.Pool`. Flint allocates a new reader per connection.
2. **Header parsing** — stdlib uses a zero-allocation header parser optimised for the common case. Flint uses `strings.SplitN` which allocates on every header line.
3. **Scheduler integration** — stdlib uses internal runtime hooks for network polling (`netpoll`) that are not available to user code.

### Worker pool behaviour

The worker pool is **slower** on localhost with fast handlers. This is expected and important to understand.

| Configuration | Req/sec | p99 | Notes |
|---|---|---|---|
| Naked goroutines | 2,985 | 102ms | No concurrency limit |
| Worker pool (100 workers) | 1,975 | 168ms | Channel dispatch overhead |

**Finding:** on localhost, the channel send + goroutine context switch adds measurable latency when handlers complete in under 1ms. The worker pool's value is not throughput — it is **memory stability under extreme concurrency**. With 500 concurrent clients and a 50ms handler (simulating a DB query), naked goroutines spawn 500 goroutines consuming ~4MB of stack space. The worker pool holds at 100 goroutines regardless of connection count. The advantage is resource predictability, not raw speed.

---

## Design decisions

### 1. `bufio.Reader` created once per connection, not per request

The Keep-Alive loop calls `parseRequest` on every iteration. The naive approach creates a new `bufio.NewReader(conn)` each call. The problem is that `bufio.Reader` has a 4096-byte internal buffer — on each read it may pull more bytes from the TCP stream than the current request needs, buffering the start of the next request. When the reader is thrown away, those bytes are lost. The next call reads from the raw `conn` and misses them.

The fix is to create the reader once in `handleConn` and pass it into `parseRequest` on every call. The buffer persists across the connection lifetime, carrying buffered bytes correctly between requests. This is the same pattern Go's stdlib uses internally.

### 2. Trie returns 405 instead of 404 when the path matches but the method does not

Most naive routers return 404 for any unmatched request. Flint's Trie distinguishes between two failure modes — path not found (true 404) and path found but method not registered (405 Method Not Allowed). This distinction matters for API clients — a 404 tells the client "this resource does not exist", while a 405 tells them "the resource exists but you used the wrong verb." HTTP/1.1 spec requires a 405 to include an `Allow` header listing valid methods. Flint implements the correct status code; the `Allow` header is a known missing feature.

### 3. `allowRequest` holds the mutex for the entire read-modify-write

The token bucket rate limiter uses a `map[string]*TokenBucket` protected by `sync.Mutex`. An earlier version called `getBucket` under the lock and `allowRequest` outside it. This created a race condition — two goroutines serving requests from the same IP could both read `tokens > 0`, both decrement, and both be admitted for the price of one token.

The fix is to merge the lookup, refill, and consume operations into a single function that holds the mutex throughout. This is the classic check-then-act race condition. The performance cost is negligible — the critical section is three integer operations taking nanoseconds.

### 4. Graceful shutdown uses `sync.WaitGroup` per connection, not per worker

An early implementation called `wg.Add(1)` once per worker goroutine at startup and `wg.Done()` inside `handleConn` once per connection. Since each worker handles many connections, `Done()` was called more times than `Add()`, causing a panic from a negative WaitGroup counter.

The correct model increments `wg` once per accepted connection in the accept loop and decrements once when `handleConn` returns. `wg.Wait()` in `Shutdown()` blocks until every in-flight connection is finished — exactly the semantic graceful shutdown requires.

### 5. `dispatch` returns `*Response` instead of writing directly to `net.Conn`

The original design had `router.dispatch(conn, req)` write the response internally. This meant `handleConn` had no access to the response after dispatch — it could not set the `Connection` header (keep-alive vs close) based on the request. By making `dispatch` return `*Response`, `handleConn` owns the final write step and can set transport-level headers after the handler runs. This is also better for testing — `dispatch` can be unit tested without a real `net.Conn`.

---

## What I would do to scale this to 1 million RPS

The current architecture handles roughly 3,000 req/sec on a single machine. Getting to 1M RPS requires changes at every layer:

**Transport layer — replace goroutines with epoll**

One goroutine per connection does not scale past ~100,000 concurrent connections due to memory pressure (each goroutine stack starts at 8KB). Production servers use event-driven I/O — Linux `epoll`, BSD `kqueue` — where a single thread monitors thousands of file descriptors and only wakes up when data arrives. Go's runtime already uses epoll internally but abstracts it behind goroutines. A custom `epoll`-based event loop would eliminate goroutine overhead entirely. This is how Nginx handles millions of connections on a single core.

**Parser — zero-allocation header parsing**

Every `strings.SplitN` call in the header parser allocates a new slice. At 1M RPS that is 1M allocations per second just for header parsing. The fix is a hand-written state machine parser that reads bytes directly without allocating — the same approach used by `picohttpparser` in C and Go's own `net/http` internally.

**Router — pre-compiled regex or radix tree**

The Trie is correct but traverses one node per path segment. A radix tree compresses common prefixes into single nodes, reducing traversal depth. At 1M RPS the difference between 5 node traversals and 2 becomes measurable.

**Rate limiter — Redis instead of in-process map**

The current rate limiter is in-process — it only works on a single server instance. At scale you run many server instances behind a load balancer. A client can bypass per-instance rate limits by having their requests spread across instances. Replacing the in-process map with a Redis `INCR` + `EXPIRE` command moves rate limit state to a shared store visible to all instances. Redis processes ~1M operations/sec with sub-millisecond latency, making it suitable for the hot path.

**Load balancing — consistent hashing**

With multiple server instances, rate limiting by IP requires routing the same IP to the same instance — otherwise the per-IP bucket is split across instances. Consistent hashing in the load balancer ensures requests from the same IP always reach the same server, making in-process rate limiting viable at scale without Redis.

**Connection — HTTP/2 multiplexing**

HTTP/1.1 Keep-Alive reuses the TCP connection but requests are still sequential — the next request cannot start until the previous response is complete (head-of-line blocking). HTTP/2 multiplexes multiple requests over a single TCP connection simultaneously. A client loading a page with 50 assets sends all 50 requests at once instead of sequentially. This reduces latency significantly without increasing server-side concurrency.

---

## Known limitations

- No `Transfer-Encoding: chunked` support — only `Content-Length` bodies
- No HTTP/2 — HTTP/1.1 only
- No `Allow` header on 405 responses
- Auth uses static Bearer token — no JWT signature verification
- Rate limiter buckets are never evicted from memory — long-running server with many unique IPs will grow unboundedly
- No request size limit — a malicious client can send an arbitrarily large `Content-Length`

---

## What I learned

Building from TCP up made abstract concepts concrete. HTTP is just text — `\r\n` delimited lines over a socket. A router is a Trie traversal. Middleware is the decorator pattern. Keep-Alive is a `for` loop that blocks on `bufio.Reader.ReadString`. Rate limiting is a mutex-protected counter. Graceful shutdown is a `sync.WaitGroup`.

The most valuable insight: every abstraction in `net/http` exists for a specific reason discovered through a specific bug or performance problem. Rebuilding those abstractions from scratch makes you a better user of them.

---

## License

MIT