package transport

import (
	"net"
	"sync"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/pkg/errors"

	"github.com/apache/thrift/lib/go/thrift"
)

// Open opens the named pipe with the provided path and timeout,
// returning a TTransport.
func Open(path string, timeout time.Duration) (*thrift.TSocket, error) {
	conn, err := winio.DialPipe(path, &timeout)
	if err != nil {
		return nil, errors.Wrapf(err, "dialing pipe '%s'", path)
	}
	return thrift.NewTSocketFromConnTimeout(conn, timeout), nil
}

func OpenServer(pipePath string, timeout time.Duration) (*TServerPipe, error) {
	return NewTServerPipeTimeout(pipePath, timeout)
}

// TServerPipe is a windows named pipe implementation of the
type TServerPipe struct {
	listener      net.Listener
	pipePath      string
	clientTimeout time.Duration

	// Protects the interrupted value to make it thread safe.
	mu          sync.RWMutex
	interrupted bool
}

func NewTServerPipeTimeout(pipePath string, clientTimeout time.Duration) (*TServerPipe, error) {
	return &TServerPipe{pipePath: pipePath, clientTimeout: clientTimeout}, nil
}

func (p *TServerPipe) Listen() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.IsListening() {
		return nil
	}

	l, err := winio.ListenPipe(p.pipePath, nil)
	if err != nil {
		return err
	}

	p.listener = l
	return nil
}

// IsListening returns whether the server transport is currently listening.
func (p *TServerPipe) IsListening() bool {
	return p.listener != nil
}

// Accept wraps the standard net.Listener accept to return a thrift.TTransport.
func (p *TServerPipe) Accept() (thrift.TTransport, error) {
	p.mu.RLock()
	interrupted := p.interrupted
	listener := p.listener
	p.mu.RUnlock()

	if interrupted {
		return nil, errors.New("transport interrupted")
	}

	conn, err := listener.Accept()
	if err != nil {
		return nil, thrift.NewTTransportExceptionFromError(err)
	}
	return thrift.NewTSocketFromConnTimeout(conn, p.clientTimeout), nil
}

func (p *TServerPipe) Close() error {
	defer func() {
		p.listener = nil
	}()
	if p.IsListening() {
		return p.listener.Close()
	}
	return nil
}

// Interrupt is a noop for this implementation
func (p *TServerPipe) Interrupt() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.interrupted = true
	p.Close()

	return nil
}
