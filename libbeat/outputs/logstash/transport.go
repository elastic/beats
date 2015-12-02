package logstash

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/elastic/libbeat/logp"
)

// TransportClient interfaces adds (re-)connect support to net.Conn.
type TransportClient interface {
	net.Conn
	Connect(timeout time.Duration) error
	IsConnected() bool
}

type tcpClient struct {
	hostport  string
	connected bool
	net.Conn
}

type tlsClient struct {
	tcpClient
	tls tls.Config
}

func newTCPClient(host string, defaultPort int) (*tcpClient, error) {
	return &tcpClient{hostport: fullAddress(host, defaultPort)}, nil
}

func (c *tcpClient) Connect(timeout time.Duration) error {
	if c.IsConnected() {
		_ = c.Close()
	}

	host, port, err := net.SplitHostPort(c.hostport)
	if err != nil {
		return err
	}

	// TODO: address lookup copied from logstash-forwarded. Really required?
	addresses, err := net.LookupHost(host)
	c.Conn = nil
	if err != nil {
		logp.Warn("DNS lookup failure \"%s\": %s", host, err)
		return err
	}

	// connect to random address
	// Use randomization on DNS reported addresses combined with timeout and ACKs
	// to spread potential load when starting up large number of beats using
	// lumberjack.
	//
	// RFCs discussing reasons for ignoring order of DNS records:
	// http://www.ietf.org/rfc/rfc3484.txt
	// > is specific to locality-based address selection for multiple dns
	// > records, but exists as prior art in "Choose some different ordering for
	// > the dns records" done by a client
	//
	// https://tools.ietf.org/html/rfc1794
	// > "Clients, of course, may reorder this information" - with respect to
	// > handling order of dns records in a response. address :=
	address := addresses[rand.Int()%len(addresses)]
	addressport := net.JoinHostPort(address, port)
	conn, err := net.DialTimeout("tcp", addressport, timeout)
	if err != nil {
		return err
	}

	c.Conn = conn
	c.connected = true
	return nil
}

func (c *tcpClient) IsConnected() bool {
	return c.connected
}

func (c *tcpClient) Close() error {
	err := c.Conn.Close()
	c.connected = false
	return err
}

func newTLSClient(host string, defaultPort int, tls *tls.Config) (*tlsClient, error) {
	c := tlsClient{}
	c.hostport = fullAddress(host, defaultPort)
	c.tls = *tls
	return &c, nil
}

func (c *tlsClient) Connect(timeout time.Duration) error {
	host, _, err := net.SplitHostPort(c.hostport)
	if err != nil {
		return err
	}

	if err := c.tcpClient.Connect(timeout); err != nil {
		return c.onFail(err)
	}

	tlsconfig := c.tls
	tlsconfig.ServerName = host
	socket := tls.Client(c.Conn, &tlsconfig)
	if err := socket.SetDeadline(time.Now().Add(timeout)); err != nil {
		_ = socket.Close()
		return c.onFail(err)
	}
	if err := socket.Handshake(); err != nil {
		_ = socket.Close()
		return c.onFail(err)
	}

	c.Conn = socket
	c.connected = true
	return nil
}

func (c *tlsClient) onFail(err error) error {
	logp.Err("SSL client failed to connect with: %v", err)
	c.Conn = nil
	c.connected = false
	return err
}

func fullAddress(host string, defaultPort int) string {
	if _, _, err := net.SplitHostPort(host); err == nil {
		return host
	}

	idx := strings.Index(host, ":")
	if idx >= 0 {
		// IPv6 address detected
		return fmt.Sprintf("[%v]:%v", host, defaultPort)
	}
	return fmt.Sprintf("%v:%v", host, defaultPort)
}
