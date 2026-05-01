package main

import (
	"flint/server"
	"fmt"
	"log"
)

func main() {
	cfg := server.Config{
		Port: ":8080",
	}

	s := server.New(cfg)

	s.GET("/ping", func(req *server.Request, res *server.Response) {
		res.Status(200).Body("pong")
	})

	s.GET("/users/:id", func(req *server.Request, res *server.Response) {
		id := req.Params["id"]
		res.Status(200).Body(fmt.Sprintf("user id is %s", id))
	})

	s.POST("/echo", func(req *server.Request, res *server.Response) {
		res.Status(200).Body(string(req.Body))
	})

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}
