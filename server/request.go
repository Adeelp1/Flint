package server

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Request holds the parsed fields of a single HTTP/1.1 request.
// It is created by parseRequest and passed to every HandlerFunc and middleware.
// Params is populated by the router when the path contains wildcard segments.
// RemoteAddr is populated by handleConn from conn.RemoteAddr().
type Request struct {
	Method     string
	Path       string
	Version    string
	Headers    map[string]string
	Body       []byte
	Params     map[string]string
	RemoteAddr string
}

func parseRequestLine(line string) (method, path, version string, err error) {
	line = strings.TrimRight(line, "\r\n")
	parts := strings.SplitN(line, " ", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid request line: %q", line)
	}
	return parts[0], parts[1], parts[2], nil
}

func parseHeader(reader *bufio.Reader) (map[string]string, error) {
	// We must read all headers before writing a response
	// otherwise the client gets confused
	// We stop when we see a blank line (\r\n)
	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("header read error: %w", err)
		}

		// blank line means headers are done
		if line == "\r\n" {
			break
		}

		line = strings.TrimRight(line, "\r\n")
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			fmt.Printf("invalid header line: %q\n", line)
			continue
		}

		headers[parts[0]] = parts[1]

	}

	return headers, nil
}

func parseBody(reader *bufio.Reader, contentLengthStr string) ([]byte, error) {
	contentLength, err := strconv.Atoi(contentLengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid Content-Length: %w", err)
	}
	body := make([]byte, contentLength)
	_, err = io.ReadFull(reader, body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	return body, nil
}

func parseRequest(reader *bufio.Reader) (*Request, error) {

	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	method, path, version, err := parseRequestLine(requestLine)
	if err != nil {
		return nil, fmt.Errorf("request line parse error: %w", err)
	}

	headers, err := parseHeader(reader)
	if err != nil {
		return nil, fmt.Errorf("header parse error: %w", err)
	}

	body := []byte{}
	if contentLengthStr, ok := headers["Content-Length"]; ok {
		body, err = parseBody(reader, contentLengthStr)
		if err != nil {
			return nil, fmt.Errorf("body parse error: %w", err)
		}
	}

	return &Request{
		Method:  method,
		Path:    path,
		Version: version,
		Headers: headers,
		Body:    body,
	}, nil
}
