package handler

import "flint/server"

func HomeHandler(req *server.Request) []byte {
	return []byte("Welcome to Flint!")
}
