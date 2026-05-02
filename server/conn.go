package server

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

func handleConn(conn net.Conn, router *Router) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		conn.SetDeadline(time.Now().Add(5 * time.Second))

		req, err := parseRequest(reader)
		if err != nil {
			// just close EOF silently
			if err == io.EOF {
				return
			}

			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return
			}
			fmt.Println("Error parsing request:", err)
			return
		}
		req.RemoteAddr = conn.RemoteAddr().String()

		// hand off to the router — it finds the handler and writes the response
		res := router.dispatch(req)

		connection := strings.ToLower(req.Headers["Connection"])
		if connection == "close" {
			res.Header("Connection", "close")
			res.write(conn)
			return
		}
		res.Header("Connection", "keep-alive")
		res.write(conn)
	}
}
