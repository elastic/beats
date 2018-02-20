package tcp

import (
	"bufio"
	"bytes"
	"net"
	"sync"

	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/filebeat/harvester"
)

// Harvester represent a TCP server
type Harvester struct {
	sync.RWMutex
	forwarder *harvester.Forwarder
	config    *config
	server    net.Listener
	clients   map[*Client]struct{}
	wg        sync.WaitGroup
	done      chan struct{}
	splitFunc bufio.SplitFunc
}

// NewHarvester creates a new harvester that will forward events
func NewHarvester(
	forwarder *harvester.Forwarder,
	config *config,
) (*Harvester, error) {

	server, err := net.Listen("tcp", config.Host)
	if err != nil {
		return nil, err
	}

	sf := splitFunc([]byte(config.LineDelimiter))
	return &Harvester{
		config:    config,
		forwarder: forwarder,
		clients:   make(map[*Client]struct{}, 0),
		done:      make(chan struct{}),
		server:    server,
		splitFunc: sf,
	}, nil
}

// Run start and run a new TCP listener to receive new data
func (h *Harvester) Run() error {
	logp.Info("Started listening for incoming TCP connection on: %s", h.config.Host)
	for {
		conn, err := h.server.Accept()
		if err != nil {
			select {
			case <-h.done:
				return nil
			default:
				logp.Debug("tcp", "Can not accept the connection: %s", err)
				continue
			}
		}

		client := NewClient(
			conn,
			h.forwarder,
			h.splitFunc,
			h.config.MaxMessageSize,
			h.config.Timeout,
		)
		logp.Debug(
			"tcp",
			"New client, remote: %s (total clients: %d)",
			conn.RemoteAddr(),
			h.clientsCount(),
		)
		h.wg.Add(1)
		go func() {
			defer conn.Close()

			h.registerClient(client)
			err := client.Handle()
			if err != nil {
				logp.Debug("tcp", "Client error: %s", err)
			}
			h.unregisterClient(client)
			logp.Debug(
				"tcp",
				"Client disconnected, remote: %s (total clients: %d)",
				conn.RemoteAddr(),
				h.clientsCount(),
			)
			h.wg.Done()
		}()
	}
}

// Stop stops accepting new incoming TCP connection and close any active clients
func (h *Harvester) Stop() {
	logp.Info("Stopping TCP harvester")
	close(h.done)
	h.server.Close()

	logp.Debug("tcp", "Closing remote connections")
	for _, client := range h.allClients() {
		client.Close()
	}
	h.wg.Wait()
	logp.Debug("tcp", "Remote connections closed")
}

func (h *Harvester) registerClient(client *Client) {
	h.Lock()
	defer h.Unlock()
	h.clients[client] = struct{}{}
}

func (h *Harvester) unregisterClient(client *Client) {
	h.Lock()
	defer h.Unlock()
	delete(h.clients, client)
}

func (h *Harvester) allClients() []*Client {
	h.RLock()
	defer h.RUnlock()
	currentClients := make([]*Client, len(h.clients))
	idx := 0
	for client := range h.clients {
		currentClients[idx] = client
		idx++
	}
	return currentClients
}

func (h *Harvester) clientsCount() int {
	h.RLock()
	defer h.RUnlock()
	return len(h.clients)
}

func splitFunc(lineDelimiter []byte) bufio.SplitFunc {
	ld := []byte(lineDelimiter)
	if bytes.Equal(ld, []byte("\n")) {
		// This will work for most usecases and will also strip \r if present.
		// CustomDelimiter, need to match completely and the delimiter will be completely removed from the
		// returned byte slice
		return bufio.ScanLines
	}
	return scanDelimiter(ld)
}
