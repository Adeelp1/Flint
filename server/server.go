package server

import (
	"fmt"
	"net"
)

type Config struct {
	Port string
}

type Server struct {
	config   Config
	listener net.Listener
}

func New(cfg Config) *Server {
	// Initialize the server with the provided configuration
	return &Server{
		config: cfg,
	}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.config.Port)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.Port, err)
	}
	s.listener = listener
	defer s.listener.Close()

	fmt.Printf("Flint listening on %s\n", s.config.Port)

	// Start accepting connections
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			fmt.Println("accept error:", err)
			continue
		}
		// Handle the connection (e.g., in a separate goroutine)
		go handleConn(conn)
	}
}
