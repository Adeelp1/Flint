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
	router   *Router
}

func New(cfg Config) *Server {
	// Initialize the server with the provided configuration
	return &Server{
		config: cfg,
		router: NewRouter(),
	}
}

func (s *Server) GET(path string, handler HandlerFunc) {
	s.router.Add("GET", path, handler)
}

func (s *Server) POST(path string, handler HandlerFunc) {
	s.router.Add("POST", path, handler)
}

func (s *Server) DELETE(path string, handler HandlerFunc) {
	s.router.Add("DELETE", path, handler)
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
		go handleConn(conn, s.router)
	}
}
