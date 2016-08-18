package v2

import (
	"errors"
	"net"

	"github.com/elastic/go-lumber/lj"
	"github.com/elastic/go-lumber/server/internal"
)

// Server serves multiple lumberjack clients supporting protocol version 2.
type Server struct {
	s *internal.Server
}

var (
	// ErrProtocolError is returned if an protocol error was detected in the
	// conversation with lumberjack server.
	ErrProtocolError = errors.New("lumberjack protocol error")
)

// NewWithListener creates a new Server using an existing net.Listener.
func NewWithListener(l net.Listener, opts ...Option) (*Server, error) {
	return newServer(opts, func(cfg internal.Config) (*internal.Server, error) {
		return internal.NewWithListener(l, cfg)
	})
}

// ListenAndServeWith uses binder to create a listener for establishing a lumberjack
// endpoint.
func ListenAndServeWith(
	binder func(network, addr string) (net.Listener, error),
	addr string,
	opts ...Option,
) (*Server, error) {
	return newServer(opts, func(cfg internal.Config) (*internal.Server, error) {
		return internal.ListenAndServeWith(binder, addr, cfg)
	})
}

// ListenAndServe listens on the TCP network address addr and handles batch
// requests from accepted lumberjack clients.
func ListenAndServe(addr string, opts ...Option) (*Server, error) {
	return newServer(opts, func(cfg internal.Config) (*internal.Server, error) {
		return internal.ListenAndServe(addr, cfg)
	})
}

// ReceiveChan returns a channel all received batch requests will be made
// available on. Batches read from channel must be ACKed.
func (s *Server) ReceiveChan() <-chan *lj.Batch {
	return s.s.ReceiveChan()
}

// Receive returns the next received batch from the receiver channel.
// Batches returned by Receive must be ACKed.
func (s *Server) Receive() *lj.Batch {
	return s.s.Receive()
}

// Close stops the listener, closes all active connections and closes the
// receiver channel returned from ReceiveChan().
func (s *Server) Close() error {
	return s.s.Close()
}

func newServer(
	opts []Option,
	mk func(cfg internal.Config) (*internal.Server, error),
) (*Server, error) {
	o, err := applyOptions(opts)
	if err != nil {
		return nil, err
	}

	mkRW := func(client net.Conn) (internal.BatchReader, internal.ACKWriter, error) {
		r := newReader(client, o.timeout, o.decoder)
		w := newWriter(client, o.timeout)
		return r, w, nil
	}

	cfg := internal.Config{
		TLS:     o.tls,
		Handler: internal.DefaultHandler(o.keepalive, mkRW),
		Channel: o.ch,
	}

	s, err := mk(cfg)
	return &Server{s}, err
}
