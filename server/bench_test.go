package server

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"testing"
)

// ─── fake reader helper ────────────────────────────────────────────────────
// creates a bufio.Reader from a raw HTTP request string
// used to feed requests into the parser without a real TCP connection

func fakeReader(raw string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(raw))
}

// ─── sample requests ───────────────────────────────────────────────────────

const getRequest = "GET /ping HTTP/1.1\r\n" +
	"Host: localhost:8443\r\n" +
	"Connection: keep-alive\r\n" +
	"\r\n"

const postRequest = "POST /echo HTTP/1.1\r\n" +
	"Host: localhost:8443\r\n" +
	"Content-Type: application/json\r\n" +
	"Content-Length: 26\r\n" +
	"Connection: keep-alive\r\n" +
	"\r\n" +
	`{"message":"hello flint!"}`

const paramRequest = "GET /users/42 HTTP/1.1\r\n" +
	"Host: localhost:8443\r\n" +
	"Authorization: Bearer 123456789abcdef\r\n" +
	"Connection: keep-alive\r\n" +
	"\r\n"

// ─── parser benchmarks ─────────────────────────────────────────────────────

// BenchmarkParseGETRequest measures how fast the parser handles a simple GET
// Run with: go test -bench=BenchmarkParseGETRequest -benchmem ./benchmark/
func BenchmarkParseGETRequest(b *testing.B) {
	b.ReportAllocs() // report memory allocations per operation

	for i := 0; i < b.N; i++ {
		reader := fakeReader(getRequest)
		_, err := parseRequest(reader)
		if err != nil {
			b.Fatal("parse error:", err)
		}
	}
}

// BenchmarkParsePOSTRequest measures parser performance with a body
func BenchmarkParsePOSTRequest(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := fakeReader(postRequest)
		_, err := parseRequest(reader)
		if err != nil {
			b.Fatal("parse error:", err)
		}
	}
}

// BenchmarkParseRequestWithAuthHeader measures header map allocation cost
func BenchmarkParseRequestWithAuthHeader(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := fakeReader(paramRequest)
		_, err := parseRequest(reader)
		if err != nil {
			b.Fatal("parse error:", err)
		}
	}
}

// ─── router benchmarks ─────────────────────────────────────────────────────

// BenchmarkRouterStaticPath measures Trie lookup for an exact path match
// Run with: go test -bench=BenchmarkRouterStaticPath -benchmem ./benchmark/
func BenchmarkRouterStaticPath(b *testing.B) {
	router := newTestRouter()

	b.ReportAllocs()
	b.ResetTimer() // don't count router setup time

	for i := 0; i < b.N; i++ {
		req := &Request{Method: "GET", Path: "/ping"}
		router.dispatch(req)
	}
}

// BenchmarkRouterDynamicPath measures Trie lookup with wildcard param extraction
// This is slower than static because it must extract and store the param value
func BenchmarkRouterDynamicPath(b *testing.B) {
	router := newTestRouter()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := &Request{Method: "GET", Path: "/users/42"}
		router.dispatch(req)
	}
}

// BenchmarkRouterNotFound measures the 404 path — no match in Trie
func BenchmarkRouterNotFound(b *testing.B) {
	router := newTestRouter()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := &Request{Method: "GET", Path: "/nonexistent/deep/path"}
		router.dispatch(req)
	}
}

// ─── middleware benchmarks ─────────────────────────────────────────────────

// BenchmarkLoggerMiddleware measures the overhead Logger adds per request
// The difference between this and the raw handler is the Logger's cost
func BenchmarkLoggerMiddleware(b *testing.B) {
	handler := Logger(io.Discard)(func(req *Request, res *Response) {
		res.Status(200).Body("pong")
	})

	req := &Request{Method: "GET", Path: "/ping"}
	res := newResponse()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		handler(req, res)
	}
}

// BenchmarkRateLimiter measures token bucket check cost under no contention
func BenchmarkRateLimiter(b *testing.B) {
	handler := RateLimit(func(req *Request, res *Response) {
		res.Status(200).Body("pong")
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// use a unique IP per iteration to avoid bucket exhaustion affecting results
		req := &Request{
			Method:     "GET",
			Path:       "/ping",
			RemoteAddr: fmt.Sprintf("192.168.%d.%d:1234", i/255, i%255),
		}
		res := newResponse()
		handler(req, res)
	}
}

// BenchmarkFullMiddlewareChain measures Logger + Auth + RateLimit combined
// Compare this to BenchmarkLoggerMiddleware to see Auth + RateLimit overhead
func BenchmarkFullMiddlewareChain(b *testing.B) {
	handler := Chain(
		func(req *Request, res *Response) {
			res.Status(200).Body("pong")
		},
		Logger(io.Discard),
		Auth("123456789abcdef"),
		RateLimit,
	)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := &Request{
			Method:     "GET",
			Path:       "/ping",
			RemoteAddr: "10.0.0.1:5000",
			Headers: map[string]string{
				"Authorization": "Bearer 123456789abcdef",
				"Connection":    "keep-alive",
			},
		}
		res := newResponse()
		handler(req, res)
	}
}

// ─── helpers ───────────────────────────────────────────────────────────────

// newTestRouter creates a router with the same routes as main.go
// used by router benchmarks above
func newTestRouter() *Router {
	r := NewRouter()
	r.add("GET", "/ping", func(req *Request, res *Response) {
		res.Status(200).Body("pong")
	})
	r.add("GET", "/users/:id", func(req *Request, res *Response) {
		res.Status(200).Body("user: " + req.Params["id"])
	})
	r.add("POST", "/echo", func(req *Request, res *Response) {
		res.Status(200).Body(string(req.Body))
	})
	return r
}
