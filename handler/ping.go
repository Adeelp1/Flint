package handler

import "flint/server"

func PingHandler(req *server.Request) []byte {
	return []byte("Pong!")
}
