package server

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type MiddlewareFunc func(next HandlerFunc) HandlerFunc

type TokenBucket struct {
	tokens         int
	capacity       int
	refillRate     int
	lastRefillTime time.Time
}

var buckets = make(map[string]*TokenBucket)
var mu sync.Mutex

func allowRequest(ip string) bool {
	mu.Lock()
	defer mu.Unlock()

	bucket, exists := buckets[ip]
	if !exists {
		bucket = &TokenBucket{
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

// for every request that passes through it
func Logger(next HandlerFunc) HandlerFunc {

	return func(req *Request, res *Response) {
		start := time.Now()

		next(req, res)

		fmt.Printf("%s %s %d %vms\n", req.Method, req.Path, res.StatusCode(), time.Since(start))
	}
}

func Auth(tocken string) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(req *Request, res *Response) {
			authHeader := req.Headers["Authorization"]
			if authHeader == "" {
				res.Status(401).Body("Unauthorized")
				return
			}

			provided := strings.TrimPrefix(authHeader, "Bearer ")
			if provided != tocken {
				res.Status(401).Body("Unauthorized")
				return
			}
			next(req, res)
		}
	}
}

func RateLimit(next HandlerFunc) HandlerFunc {
	// This is a very naive implementation of rate limiting
	// In production, you would want to use a more robust solution
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

func Chain(handler HandlerFunc, middlewares ...MiddlewareFunc) HandlerFunc {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
