package udp

import (
	"net"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/elastic/beats/libbeat/logp"
)

// Config options for the UDPServer
type Config struct {
	Host           string        `config:"host"`
	MaxMessageSize int           `config:"max_message_size" validate:"positive,nonzero"`
	Timeout        time.Duration `config:"timeout"`
}

// Server creates a simple UDP Server and listen to a specific host:port and will send any
// event received to the callback method.
type Server struct {
	running  atomic.Bool
	config   *Config
	callback func(data []byte, addr net.Addr)
	listener net.PacketConn
	log      *logp.Logger
	wg       sync.WaitGroup
}

// New returns a new UDPServer instance.
func New(config *Config, callback func(data []byte, addr net.Addr)) *Server {
	return &Server{
		running:  atomic.MakeBool(false),
		config:   config,
		callback: callback,
		log:      logp.NewLogger("udp").With("address", config.Host),
	}
}

// Start starts the UDP Server and listen to incoming events.
func (u *Server) Start() error {
	var err error

	u.listener, err = net.ListenPacket("udp", u.config.Host)
	if err != nil {
		return err
	}
	defer u.listener.Close()
	// Mostly used in tests, instead of relying on sleep.
	u.running.Swap(true)
	defer u.running.Swap(false)

	u.wg.Add(1)
	defer u.wg.Done()

	u.log.Info("Started listening for UDP connection")

	for u.IsRunning() {
		buffer := make([]byte, u.config.MaxMessageSize)
		u.listener.SetDeadline(time.Now().Add(u.config.Timeout))
		length, addr, err := u.listener.ReadFrom(buffer)

		if err != nil {
			// don't log any deadline events.
			e, ok := err.(net.Error)
			if ok && e.Timeout() {
				continue
			}
			u.log.Errorw("Error reading from the socket", "error", err)
			continue
		}

		if length > 0 {
			u.callback(buffer[:length], addr)
		}
	}
	return nil
}

// IsRunning returns true if the UDP server is accepting new connection.
func (u *Server) IsRunning() bool {
	return u.running.Load()
}

// Stop stops the current udp server.
func (u *Server) Stop() {
	u.log.Info("Stopping UDP server")
	u.listener.Close()
	u.running.Swap(false)
	u.wg.Wait()
	u.log.Info("UDP server stopped")
}
