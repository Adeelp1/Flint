package server

import (
	"bufio"
	"fmt"
	"net"
)

func handleConn(conn net.Conn) {
	defer conn.Close()

	_, err := readRequest(conn)
	if err != nil {
		return
	}

	err = writeResponse(conn)
	if err != nil {
		fmt.Println("Error writing response:", err)
	}
}

func readRequest(conn net.Conn) (string, error) {
	reader := bufio.NewReader(conn)

	requestLine, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("read error:", err)
		return "", err
	}

	fmt.Printf("raw request line: %q\n", requestLine)

	// We must read all headers before writing a response
	// otherwise the client gets confused
	// We stop when we see a blank line (\r\n)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("header read error:", err)
			return "", err
		}
		// blank line means headers are done
		if line == "\r\n" {
			break
		}
	}

	return requestLine, nil
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
