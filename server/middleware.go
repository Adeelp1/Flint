package server

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// MiddlewareFunc is a function that wraps a HandlerFunc with additional logic.
// Middleware runs before and/or after the next handler in the chain.
// Use Chain() to compose multiple middleware functions together.
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// Chain applies a list of middleware to a handler, returning a new HandlerFunc.
// Middleware is applied left to right — the first in the list is outermost.
// Example: Chain(h, Logger, Auth(token), RateLimit)
// Request order: Logger → Auth → RateLimit → h
func Chain(handler HandlerFunc, middlewares ...MiddlewareFunc) HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

type tokenBucket struct {
	tokens         int
	capacity       int
	refillRate     int
	lastRefillTime time.Time
}

// buckets stores one tokenBucket per unique IP address.
// entries are never evicted — memory grows with unique IP count.
// for production use, replace with a Redis-backed store with TTL expiry.
var buckets = make(map[string]*tokenBucket)
var mu sync.Mutex

func allowRequest(ip string) bool {
	mu.Lock()
	defer mu.Unlock()

	bucket, exists := buckets[ip]
	if !exists {
		bucket = &tokenBucket{
			tokens:         10,
			capacity:       10,
			refillRate:     5,
			lastRefillTime: time.Now(),
		}
		buckets[ip] = bucket
	}

	// refill
	elapsed := time.Since(bucket.lastRefillTime).Seconds()
	tokensToAdd := int(elapsed) * bucket.refillRate
	if tokensToAdd > 0 {
		bucket.tokens = min(bucket.capacity, bucket.tokens+tokensToAdd)
		bucket.lastRefillTime = time.Now()
	}

	// consume
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}
	return false
}

// Logger returns a middleware that writes method, path, status code,
// and request duration to out after each request completes.
// Pass os.Stdout for development, io.Discard for benchmarks,
// or any io.Writer for custom log destinations.
func Logger(out io.Writer) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(req *Request, res *Response) {
			start := time.Now()

			next(req, res)

			fmt.Fprintf(out, "%s %s %d %v\n", req.Method, req.Path, res.StatusCode(), time.Since(start))
		}
	}
}

// Auth returns a middleware that validates the Authorization header against
// the provided Bearer token. Returns 401 Unauthorized if the token is
// missing or does not match. Token is configured via Config.AuthToken.
func Auth(token string) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(req *Request, res *Response) {
			authHeader := req.Headers["Authorization"]
			if authHeader == "" {
				res.Status(401).Body("Unauthorized")
				return
			}

			provided := strings.TrimPrefix(authHeader, "Bearer ")
			if provided != token {
				res.Status(401).Body("Unauthorized")
				return
			}
			next(req, res)
		}
	}
}

// RateLimit is a middleware that enforces per-IP request limits using a
// token bucket algorithm. Each IP gets 10 tokens refilling at 5 per second.
// Returns 429 Too Many Requests when the bucket is empty.
// The token bucket map is protected by a sync.Mutex for concurrent safety.
func RateLimit(next HandlerFunc) HandlerFunc {
	return func(req *Request, res *Response) {
		ip := req.RemoteAddr
		if ip == "" {
			ip = "unknown"
		}

		if !allowRequest(ip) {
			res.Status(429).Body("Too Many Requests")
			return
		}

		next(req, res)
	}

}
