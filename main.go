package main

import (
	"flint/handler"
	"flint/server"
	"log"
)

func main() {
	cfg := server.Config{
		Port:       ":8080",
		AuthTocken: "123456789abcdef",
	}

	s := server.New(cfg)

	s.GET("/ping", server.Chain(handler.PingHandler, server.Logger, server.RateLimit))

	s.GET("/users/:id", server.Chain(handler.HomeHandler, server.Logger, server.Auth(cfg.AuthTocken), server.RateLimit))

	s.POST("/echo", server.Chain(handler.EchoHandler, server.Logger, server.Auth(cfg.AuthTocken), server.RateLimit))

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}
