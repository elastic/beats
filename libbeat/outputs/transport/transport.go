package transport

import (
	"errors"
	"net"

	"github.com/elastic/beats/libbeat/logp"
)

type Dialer interface {
	Dial(network, address string) (net.Conn, error)
}

type DialerFunc func(network, address string) (net.Conn, error)

var (
	ErrNotConnected = errors.New("client is not connected")

	debugf = logp.MakeDebug("transport")
)

func (d DialerFunc) Dial(network, address string) (net.Conn, error) {
	return d(network, address)
}

func Dial(c *Config, network, address string) (net.Conn, error) {
	d, err := MakeDialer(c)
	if err != nil {
		return nil, err
	}
	return d.Dial(network, address)
}
