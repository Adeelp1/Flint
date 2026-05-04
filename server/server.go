package server

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync"
)

// Config holds the startup configuration for a Flint server instance.
// Port is the address to listen on (e.g. ":8443").
// AuthToken is the Bearer token required for protected routes.
type Config struct {
	Port      string
	AuthToken string
}

// Server is the core Flint server. It owns the TLS listener, worker pool,
// router, and WaitGroup used for graceful shutdown.
// Create one with New(cfg) and start it with Start().
type Server struct {
	config   Config
	listener net.Listener
	router   *Router
	wg       sync.WaitGroup
}

// New creates and returns a new Server initialised with the provided Config.
// The router is empty at creation — register routes with GET, POST, DELETE
// before calling Start().
func New(cfg Config) *Server {
	return &Server{
		config: cfg,
		router: NewRouter(),
	}
}

// GET registers a HandlerFunc for GET requests to the given path.
// Path segments prefixed with : are treated as wildcard params
// e.g. "/users/:id" captures the value as req.Params["id"].
func (s *Server) GET(path string, handler HandlerFunc) {
	s.router.add("GET", path, handler)
}

// POST registers a HandlerFunc for POST requests to the given path.
func (s *Server) POST(path string, handler HandlerFunc) {
	s.router.add("POST", path, handler)
}

// DELETE registers a HandlerFunc for DELETE requests to the given path.
func (s *Server) DELETE(path string, handler HandlerFunc) {
	s.router.add("DELETE", path, handler)
}

// Start loads the TLS certificate, binds the port, starts the worker pool,
// and enters the accept loop. It blocks until the listener is closed.
// Call Shutdown() from another goroutine to stop the server gracefully.
func (s *Server) Start() error {
	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		return fmt.Errorf("failed to load TLS certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	listener, err := tls.Listen("tcp", s.config.Port, tlsConfig)
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
				s.wg.Done()
			}
		}(i)
	}

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			close(connChan)
			return fmt.Errorf("accept error: %w", err)
		}
		s.wg.Add(1)
		// Handle the connection (e.g., in a separate goroutine)
		connChan <- conn
	}
}

// Shutdown closes the listener to stop accepting new connections, then
// blocks until all in-flight connections have finished via sync.WaitGroup.
// Call this after receiving an OS signal (SIGINT, SIGTERM).
func (s *Server) Shutdown() {
	fmt.Println("Shutting down.....")

	s.listener.Close()

	s.wg.Wait()

	fmt.Println("Server stopped")
}
