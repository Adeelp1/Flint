package server

import (
	"fmt"
	"net"
	"strings"
)

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

func (r *Response) StatusCode() int {
	return r.statusCode
}

func (r *Response) Status(code int) *Response {
	r.statusCode = code
	return r
}

func (r *Response) Body(body string) *Response {
	r.body = body
	r.headers["Content-Length"] = fmt.Sprintf("%d", len(body))
	return r
}

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
