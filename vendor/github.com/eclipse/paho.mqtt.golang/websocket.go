package mqtt

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebsocketOptions are config options for a websocket dialer
type WebsocketOptions struct {
	ReadBufferSize  int
	WriteBufferSize int
	Proxy           ProxyFunction
}

type ProxyFunction func(req *http.Request) (*url.URL, error)

// NewWebsocket returns a new websocket and returns a net.Conn compatible interface using the gorilla/websocket package
func NewWebsocket(host string, tlsc *tls.Config, timeout time.Duration, requestHeader http.Header, options *WebsocketOptions) (net.Conn, error) {
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	if options == nil {
		// Apply default options
		options = &WebsocketOptions{}
	}
	if options.Proxy == nil {
		options.Proxy = http.ProxyFromEnvironment
	}
	dialer := &websocket.Dialer{
		Proxy:             options.Proxy,
		HandshakeTimeout:  timeout,
		EnableCompression: false,
		TLSClientConfig:   tlsc,
		Subprotocols:      []string{"mqtt"},
		ReadBufferSize:    options.ReadBufferSize,
		WriteBufferSize:   options.WriteBufferSize,
	}

	ws, resp, err := dialer.Dial(host, requestHeader)

	if err != nil {
		if resp != nil {
			WARN.Println(CLI, fmt.Sprintf("Websocket handshake failure. StatusCode: %d. Body: %s", resp.StatusCode, resp.Body))
		}
		return nil, err
	}

	wrapper := &websocketConnector{
		Conn: ws,
	}
	return wrapper, err
}

// websocketConnector is a websocket wrapper so it satisfies the net.Conn interface so it is a
// drop in replacement of the golang.org/x/net/websocket package.
// Implementation guide taken from https://github.com/gorilla/websocket/issues/282
type websocketConnector struct {
	*websocket.Conn
	r   io.Reader
	rio sync.Mutex
	wio sync.Mutex
}

// SetDeadline sets both the read and write deadlines
func (c *websocketConnector) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	err := c.SetWriteDeadline(t)
	return err
}

// Write writes data to the websocket
func (c *websocketConnector) Write(p []byte) (int, error) {
	c.wio.Lock()
	defer c.wio.Unlock()

	err := c.WriteMessage(websocket.BinaryMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Read reads the current websocket frame
func (c *websocketConnector) Read(p []byte) (int, error) {
	c.rio.Lock()
	defer c.rio.Unlock()
	for {
		if c.r == nil {
			// Advance to next message.
			var err error
			_, c.r, err = c.NextReader()
			if err != nil {
				return 0, err
			}
		}
		n, err := c.r.Read(p)
		if err == io.EOF {
			// At end of message.
			c.r = nil
			if n > 0 {
				return n, nil
			}
			// No data read, continue to next message.
			continue
		}
		return n, err
	}
}
