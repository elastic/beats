package udp

import (
	"net"
	"sync"
	"time"

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
	config   *Config
	callback func(data []byte, addr net.Addr)
	Listener net.PacketConn
	log      *logp.Logger
	wg       sync.WaitGroup
	done     chan struct{}
}

// New returns a new UDPServer instance.
func New(config *Config, callback func(data []byte, addr net.Addr)) *Server {
	return &Server{
		config:   config,
		callback: callback,
		log:      logp.NewLogger("udp").With("address", config.Host),
		done:     make(chan struct{}),
	}
}

// Start starts the UDP Server and listen to incoming events.
func (u *Server) Start() error {
	var err error
	u.Listener, err = net.ListenPacket("udp", u.config.Host)
	if err != nil {
		return err
	}
	u.log.Info("Started listening for UDP connection")
	u.wg.Add(1)
	go func() {
		defer u.wg.Done()
		u.run()
	}()
	return nil
}

func (u *Server) run() {
	for {
		select {
		case <-u.done:
			return
		default:
		}

		buffer := make([]byte, u.config.MaxMessageSize)
		u.Listener.SetDeadline(time.Now().Add(u.config.Timeout))
		length, addr, err := u.Listener.ReadFrom(buffer)

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
}

// Stop stops the current udp server.
func (u *Server) Stop() {
	u.log.Info("Stopping UDP server")
	u.Listener.Close()
	close(u.done)
	u.wg.Wait()
	u.log.Info("UDP server stopped")
}
