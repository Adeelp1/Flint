package main

import (
	"flint/server"
	"fmt"
	"log"
)

func main() {
	cfg := server.Config{
		Port:       ":8080",
		AuthTocken: "123456789abcdef",
	}

	s := server.New(cfg)

	s.GET("/ping", server.Chain(func(req *server.Request, res *server.Response) {
		res.Status(200).Body("pong")
	}, server.Logger, server.RateLimit))

	s.GET("/users/:id", server.Chain(func(req *server.Request, res *server.Response) {
		id := req.Params["id"]
		res.Status(200).Body(fmt.Sprintf("user id is %s", id))
	}, server.Logger, server.Auth(cfg.AuthTocken), server.RateLimit))

	s.POST("/echo", server.Chain(func(req *server.Request, res *server.Response) {
		res.Status(200).Body(string(req.Body))
	}, server.Logger, server.Auth(cfg.AuthTocken), server.RateLimit))

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}
