package tcp

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"sync"

	"github.com/elastic/beats/filebeat/inputsource"
	"github.com/elastic/beats/libbeat/logp"
)

// Server represent a TCP server
type Server struct {
	sync.RWMutex
	callback  inputsource.NetworkFunc
	config    *Config
	Listener  net.Listener
	clients   map[*client]struct{}
	wg        sync.WaitGroup
	done      chan struct{}
	splitFunc bufio.SplitFunc
	log       *logp.Logger
}

// New creates a new tcp server
func New(
	config *Config,
	callback inputsource.NetworkFunc,
) (*Server, error) {

	if len(config.LineDelimiter) == 0 {
		return nil, fmt.Errorf("empty line delimiter")
	}

	sf := splitFunc([]byte(config.LineDelimiter))
	return &Server{
		config:    config,
		callback:  callback,
		clients:   make(map[*client]struct{}, 0),
		done:      make(chan struct{}),
		splitFunc: sf,
		log:       logp.NewLogger("tcp").With("address", config.Host),
	}, nil
}

// Start listen to the TCP socket.
func (s *Server) Start() error {
	var err error
	s.Listener, err = net.Listen("tcp", s.config.Host)
	if err != nil {
		return err
	}

	s.log.Info("Started listening for TCP connection")

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.run()
	}()
	return nil
}

// Run start and run a new TCP listener to receive new data
func (s *Server) run() {
	for {
		conn, err := s.Listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				s.log.Debugw("Can not accept the connection", "error", err)
				continue
			}
		}

		client := newClient(
			conn,
			s.log,
			s.callback,
			s.splitFunc,
			uint64(s.config.MaxMessageSize),
			s.config.Timeout,
		)

		s.log.Debugw("New client", "address", conn.RemoteAddr(), "total", s.clientsCount())
		s.wg.Add(1)
		go func() {
			defer logp.Recover("recovering from a tcp client crash")
			defer s.wg.Done()
			defer conn.Close()

			s.registerClient(client)
			defer s.unregisterClient(client)

			err := client.handle()
			if err != nil {
				s.log.Debugw("Client error", "error", err)
			}

			s.log.Debugw("Client disconnected", "address", conn.RemoteAddr(), "total", s.clientsCount())
		}()
	}
}

// Stop stops accepting new incoming TCP connection and close any active clients
func (s *Server) Stop() {
	s.log.Info("Stopping TCP server")
	close(s.done)
	s.Listener.Close()
	for _, client := range s.allClients() {
		client.close()
	}
	s.wg.Wait()
	s.log.Info("TCP server stopped")
}

func (s *Server) registerClient(client *client) {
	s.Lock()
	defer s.Unlock()
	s.clients[client] = struct{}{}
}

func (s *Server) unregisterClient(client *client) {
	s.Lock()
	defer s.Unlock()
	delete(s.clients, client)
}

func (s *Server) allClients() []*client {
	s.RLock()
	defer s.RUnlock()
	currentClients := make([]*client, len(s.clients))
	idx := 0
	for client := range s.clients {
		currentClients[idx] = client
		idx++
	}
	return currentClients
}

func (s *Server) clientsCount() int {
	s.RLock()
	defer s.RUnlock()
	return len(s.clients)
}

func splitFunc(lineDelimiter []byte) bufio.SplitFunc {
	ld := []byte(lineDelimiter)
	if bytes.Equal(ld, []byte("\n")) {
		// This will work for most usecases and will also strip \r if present.
		// CustomDelimiter, need to match completely and the delimiter will be completely removed from
		// the returned byte slice
		return bufio.ScanLines
	}
	return factoryDelimiter(ld)
}
