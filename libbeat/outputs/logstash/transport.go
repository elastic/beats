package logstash

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"

	"github.com/elastic/beats/libbeat/logp"
)

// TransportClient interfaces adds (re-)connect support to net.Conn.
type TransportClient interface {
	net.Conn
	Connect(timeout time.Duration) error
	IsConnected() bool
}

type tcpClient struct {
	hostport  string
	proxy     *proxyConfig
	connected bool
	conn      net.Conn
	mutex     sync.Mutex
}

type tlsClient struct {
	tcpClient
	tls *tls.Config
}

func newTCPClient(host string, defaultPort int, proxy *proxyConfig) (*tcpClient, error) {
	return &tcpClient{hostport: fullAddress(host, defaultPort), proxy: proxy}, nil
}

func (c *tcpClient) Connect(timeout time.Duration) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.doConnect(timeout)
}

func (c *tcpClient) doConnect(timeout time.Duration) error {
	if c.connected {
		c.connected = false
		_ = c.conn.Close()
	}

	host, port, err := net.SplitHostPort(c.hostport)
	if err != nil {
		return err
	}

	var address string
	var dialer proxy.Dialer = &net.Dialer{Timeout: timeout}
	if c.proxy != nil && c.proxy.parsedURL != nil {
		// Do not resolve the address locally. It will be resolved on the
		// SOCKS server. The beat will have no control over the randomization
		// of the IP used when multiple IPs are returned by DNS.
		if !c.proxy.LocalResolve {
			address = host
		}

		dialer, err = proxy.FromURL(c.proxy.parsedURL, dialer)
		if err != nil {
			return err
		}
	}

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
	// > handling order of dns records in a response.orwarded. Really required?
	if address == "" {
		addresses, err := net.LookupHost(host)
		if err != nil {
			logp.Warn(`DNS lookup failure "%s": %v`, host, err)
			return err
		}
		address = addresses[rand.Int()%len(addresses)]
	}

	conn, err := dialer.Dial("tcp", net.JoinHostPort(address, port))
	if err != nil {
		return err
	}

	c.conn = conn
	c.connected = true
	return nil
}

func (c *tcpClient) IsConnected() bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.connected
}

func (c *tcpClient) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.connected {
		debug("closing")
		c.connected = false
		return c.conn.Close()
	}
	return nil
}

func (c *tcpClient) getConn() net.Conn {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.connected {
		return nil
	}
	return c.conn
}

func (c *tcpClient) Read(b []byte) (int, error) {
	conn := c.getConn()
	if conn == nil {
		return 0, ErrNotConnected
	}

	debug("try read: %v", len(b))
	n, err := conn.Read(b)
	return n, c.handleError(err)
}

func (c *tcpClient) Write(b []byte) (int, error) {
	conn := c.getConn()
	if conn == nil {
		return 0, ErrNotConnected
	}

	n, err := c.conn.Write(b)
	return n, c.handleError(err)
}

func (c *tcpClient) LocalAddr() net.Addr {
	conn := c.getConn()
	if conn != nil {
		return c.conn.LocalAddr()
	}
	return nil

}

func (c *tcpClient) RemoteAddr() net.Addr {
	conn := c.getConn()
	if conn != nil {
		return c.conn.LocalAddr()
	}
	return nil
}

func (c *tcpClient) SetDeadline(t time.Time) error {
	conn := c.getConn()
	if conn == nil {
		return ErrNotConnected
	}

	err := conn.SetDeadline(t)
	return c.handleError(err)
}

func (c *tcpClient) SetReadDeadline(t time.Time) error {
	conn := c.getConn()
	if conn == nil {
		return ErrNotConnected
	}

	err := conn.SetReadDeadline(t)
	return c.handleError(err)
}

func (c *tcpClient) SetWriteDeadline(t time.Time) error {
	conn := c.getConn()
	if conn == nil {
		return ErrNotConnected
	}

	err := conn.SetWriteDeadline(t)
	return c.handleError(err)
}

func (c *tcpClient) handleError(err error) error {
	if err != nil {
		debug("handle error: %v", err)

		if nerr, ok := err.(net.Error); !(ok && (nerr.Temporary() || nerr.Timeout())) {
			c.Close()
		}
	}
	return err
}

func newTLSClient(host string, defaultPort int, tls *tls.Config, proxy *proxyConfig) (*tlsClient, error) {
	c := tlsClient{}
	c.hostport = fullAddress(host, defaultPort)
	c.tls = tls
	c.proxy = proxy
	return &c, nil
}

func (c *tlsClient) Connect(timeout time.Duration) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	host, _, err := net.SplitHostPort(c.hostport)
	if err != nil {
		return err
	}

	if err := c.tcpClient.doConnect(timeout); err != nil {
		return c.onFail(err)
	}

	tlsconfig := c.tls
	tlsconfig.ServerName = host
	socket := tls.Client(c.conn, tlsconfig)
	if err := socket.SetDeadline(time.Now().Add(timeout)); err != nil {
		_ = socket.Close()
		return c.onFail(err)
	}
	if err := socket.Handshake(); err != nil {
		_ = socket.Close()
		return c.onFail(err)
	}

	c.conn = socket
	c.connected = true
	return nil
}

func (c *tlsClient) onFail(err error) error {
	logp.Err("SSL client failed to connect with: %v", err)
	c.conn = nil
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
