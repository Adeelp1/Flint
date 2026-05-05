package main

import (
	"flint/handler"
	"flint/server"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := server.Config{
		Port:      ":8443",
		AuthToken: "123456789abcdef",
	}

	s := server.New(cfg)

	s.GET("/ping", server.Chain(handler.PingHandler, server.Logger(os.Stdout), server.RateLimit))

	s.GET("/users/:id", server.Chain(handler.HomeHandler, server.Logger(os.Stdout), server.Auth(cfg.AuthToken), server.RateLimit))

	s.POST("/echo", server.Chain(handler.EchoHandler, server.Logger(os.Stdout), server.Auth(cfg.AuthToken), server.RateLimit))

	go func() {
		if err := s.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit

	s.Shutdown()
}
