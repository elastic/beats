package helper

import (
	"time"

	"github.com/elastic/beats/libbeat/outputs/transport"
)

type DialerBuilder interface {
	Stringer
	MakeDialer(time.Duration)
}

type TransportDefault struct{}

func (t *TransportDefault) MakeDialer(timeout time.Duration) (transport.Dialer, error) {
	return transport.NetDialer(t), nil
}

func (t *TransportDefault) String() string {
	return "TCP/UDP"
}

func NewTransportDefault() *TransportDefault {
	return &TransportDefault{}
}

func NewTransportNpipe(path string) *TransportNpipe {
	return &TransportNpipe{path: path}
}

// NewTransportUnix returns a new TransportUnix instance that will allow the HTTP client to communicate
// over a unix domain socket it require a valid path to the socket on the filesystem.
func NewTransportUnix(path string) *TransportUnix {
	return &TransportDefault{path: path}
}
