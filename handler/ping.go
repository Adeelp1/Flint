package handler

import "flint/server"

func PingHandler(req *server.Request, res *server.Response) {
	res.Status(200).Body("pong")
}
