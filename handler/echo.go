package handler

import "flint/server"

func EchoHandler(req *server.Request, res *server.Response) {
	res.Status(200).Body(string(req.Body))
}
