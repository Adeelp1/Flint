package server

import (
	"fmt"
	"net"
	"strings"
)

// Response holds the status code, headers, and body that will be written
// back to the client. Handlers write to it via the Status, Body, and Header
// methods. The response is flushed to the TCP connection by handleConn
// after the handler and all middleware have returned.
type Response struct {
	statusCode int
	headers    map[string]string
	body       string
}

func newResponse() *Response {
	return &Response{
		statusCode: 200,
		headers: map[string]string{
			"Content-Length": "0",
		},
	}
}

// StatusCode returns the HTTP status code currently set on the response.
// Used by Logger middleware to read the status after the handler runs.
func (r *Response) StatusCode() int {
	return r.statusCode
}

// Status sets the HTTP status code on the response and returns the response
// for method chaining: res.Status(404).Body("not found").
func (r *Response) Status(code int) *Response {
	r.statusCode = code
	return r
}

// Body sets the response body and automatically updates Content-Length.
// Returns the response for method chaining.
func (r *Response) Body(body string) *Response {
	r.body = body
	r.headers["Content-Length"] = fmt.Sprintf("%d", len(body))
	return r
}

// Header sets a single response header key-value pair.
// Returns the response for method chaining.
func (r *Response) Header(key, value string) *Response {
	r.headers[key] = value
	return r
}

func statusText(code int) string {
	switch code {
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 500:
		return "Internal Server Error"
	default:
		return "Unknown Status"
	}
}

// write flushes the response to the TCP connection
func (r *Response) write(conn net.Conn) error {
	// status line
	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", r.statusCode, statusText(r.statusCode))

	// headers
	r.headers["Content-Type"] = "text/plain"

	var headerLines strings.Builder
	for key, value := range r.headers {
		headerLines.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	// assemble full response
	raw := statusLine + headerLines.String() + "\r\n" + r.body

	_, err := conn.Write([]byte(raw))
	if err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}

	return nil
}
