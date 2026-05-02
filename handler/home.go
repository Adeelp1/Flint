package handler

import (
	"flint/server"
	"fmt"
)

func HomeHandler(req *server.Request, res *server.Response) {
	id := req.Params["id"]
	res.Status(200).Body(fmt.Sprintf("user id is %s", id))
}
