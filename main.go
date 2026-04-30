package main

import (
	"flint/server"
	"log"
)

func main() {
	cfg := server.Config{
		Port: ":8080",
	}

	s := server.New(cfg)

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}
