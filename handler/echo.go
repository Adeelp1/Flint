package handler

import "flint/server"

func EchoHandler(req *server.Request) []byte {
	return []byte("Echo: " + string(req.Body))
}
