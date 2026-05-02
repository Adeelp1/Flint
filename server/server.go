package server

import (
	"fmt"
	"net"
)

type Config struct {
	Port       string
	AuthTocken string
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

	fmt.Printf("Flint listening on %s\n", s.config.Port)

	connChan := make(chan net.Conn, 1000)

	numWorkers := 100
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			for conn := range connChan {
				handleConn(conn, s.router)
			}
		}(i)
	}

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			close(connChan)
			return fmt.Errorf("accept error: %w", err)
		}
		// Handle the connection (e.g., in a separate goroutine)
		connChan <- conn
	}
}
