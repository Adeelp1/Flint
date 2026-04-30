package server

import (
	"fmt"
	"net"
)

func handleConn(conn net.Conn) {
	defer conn.Close()

	req, err := parseRequest(conn)
	if err != nil {
		fmt.Println("Error parsing request:", err)
		return
	}

	fmt.Printf("method: %s  path: %s  version: %s\n", req.Method, req.Path, req.Version)

	err = writeResponse(conn)
	if err != nil {
		fmt.Println("Error writing response:", err)
	}
}

func writeResponse(conn net.Conn) error {
	response := "HTTP/1.1 200 OK\r\n" +
		"Content-Type: text/plain\r\n" +
		"Content-Length: 13\r\n" +
		"Connection: close\r\n" +
		"\r\n" +
		"Hello, Flint!"

	_, err := conn.Write([]byte(response))
	if err != nil {
		fmt.Println("write error:", err)
		return err
	}
	fmt.Println("response sent successfully")
	return nil
}
