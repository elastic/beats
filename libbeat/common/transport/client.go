// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package transport

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/testing"
	"github.com/elastic/elastic-agent-libs/logp"
)

type Client struct {
	log     *logp.Logger
	dialer  Dialer
	network string
	host    string
	config  Config

	conn  net.Conn
	mutex sync.Mutex
}

type Config struct {
	Proxy   *ProxyConfig
	TLS     *tlscommon.TLSConfig
	Timeout time.Duration
	Stats   IOStatser
}

func NewClient(c Config, network, host string, defaultPort int) (*Client, error) {
	// do some sanity checks regarding network and Config matching +
	// address being parseable
	switch network {
	case "tcp", "tcp4", "tcp6":
	case "udp", "udp4", "udp6":
		if c.TLS == nil && c.Proxy == nil {
			break
		}
		fallthrough
	default:
		return nil, fmt.Errorf("unsupported network type %v", network)
	}

	dialer, err := MakeDialer(c)
	if err != nil {
		return nil, err
	}

	return NewClientWithDialer(dialer, c, network, host, defaultPort)
}

func NewClientWithDialer(d Dialer, c Config, network, host string, defaultPort int) (*Client, error) {
	// check address being parseable
	host = fullAddress(host, defaultPort)
	_, _, err := net.SplitHostPort(host)
	if err != nil {
		return nil, err
	}

	client := &Client{
		log:     logp.NewLogger(logSelector),
		dialer:  d,
		network: network,
		host:    host,
		config:  c,
	}
	return client, nil
}

func (c *Client) Connect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}

	conn, err := c.dialer.Dial(c.network, c.host)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *Client) IsConnected() bool {
	c.mutex.Lock()
	b := c.conn != nil
	c.mutex.Unlock()
	return b
}

func (c *Client) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn != nil {
		c.log.Debug("closing")
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *Client) getConn() net.Conn {
	c.mutex.Lock()
	conn := c.conn
	c.mutex.Unlock()
	return conn
}

func (c *Client) Read(b []byte) (int, error) {
	conn := c.getConn()
	if conn == nil {
		return 0, ErrNotConnected
	}

	n, err := conn.Read(b)
	return n, c.handleError(err)
}

func (c *Client) Write(b []byte) (int, error) {
	conn := c.getConn()
	if conn == nil {
		return 0, ErrNotConnected
	}

	n, err := c.conn.Write(b)
	return n, c.handleError(err)
}

func (c *Client) LocalAddr() net.Addr {
	conn := c.getConn()
	if conn != nil {
		return c.conn.LocalAddr()
	}
	return nil
}

func (c *Client) RemoteAddr() net.Addr {
	conn := c.getConn()
	if conn != nil {
		return c.conn.RemoteAddr()
	}
	return nil
}

func (c *Client) Host() string {
	return c.host
}

func (c *Client) SetDeadline(t time.Time) error {
	conn := c.getConn()
	if conn == nil {
		return ErrNotConnected
	}

	err := conn.SetDeadline(t)
	return c.handleError(err)
}

func (c *Client) SetReadDeadline(t time.Time) error {
	conn := c.getConn()
	if conn == nil {
		return ErrNotConnected
	}

	err := conn.SetReadDeadline(t)
	return c.handleError(err)
}

func (c *Client) SetWriteDeadline(t time.Time) error {
	conn := c.getConn()
	if conn == nil {
		return ErrNotConnected
	}

	err := conn.SetWriteDeadline(t)
	return c.handleError(err)
}

func (c *Client) handleError(err error) error {
	if err != nil {
		c.log.Debugf("handle error: %+v", err)

		if nerr, ok := err.(net.Error); !(ok && (nerr.Temporary() || nerr.Timeout())) {
			_ = c.Close()
		}
	}
	return err
}

func (c *Client) Test(d testing.Driver) {
	d.Run("logstash: "+c.host, func(d testing.Driver) {
		d.Run("connection", func(d testing.Driver) {
			netDialer := TestNetDialer(d, c.config.Timeout)
			_, err := netDialer.Dial("tcp", c.host)
			d.Fatal("dial up", err)
		})

		if c.config.TLS == nil {
			d.Warn("TLS", "secure connection disabled")
		} else {
			d.Run("TLS", func(d testing.Driver) {
				netDialer := NetDialer(c.config.Timeout)
				tlsDialer := TestTLSDialer(d, netDialer, c.config.TLS, c.config.Timeout)
				_, err := tlsDialer.Dial("tcp", c.host)
				d.Fatal("dial up", err)
			})
		}

		err := c.Connect()
		d.Fatal("talk to server", err)
	})
}

func (c *Client) String() string {
	return c.network + "://" + c.host
}
