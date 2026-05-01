package server

import (
	"fmt"
	"io"
	"net"
)

func handleConn(conn net.Conn, router *Router) {
	defer conn.Close()

	req, err := parseRequest(conn)
	if err != nil {
		// EOF means the client connected but sent nothing — completely normal
		// browsers do this speculatively — just close silently
		if err == io.EOF {
			return
		}
		fmt.Println("Error parsing request:", err)
		return
	}

	fmt.Printf("method: %s  path: %s  version: %s\n", req.Method, req.Path, req.Version)

	// hand off to the router — it finds the handler and writes the response
	router.dispatch(conn, req)
}
