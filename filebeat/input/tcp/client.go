package tcp

import (
	"bufio"
	"net"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// Client is a remote client.
type Client struct {
	conn           net.Conn
	forwarder      *harvester.Forwarder
	done           chan struct{}
	metadata       common.MapStr
	splitFunc      bufio.SplitFunc
	maxReadMessage uint64
	timeout        time.Duration
}

// NewClient returns a new client instance for the remote connection.
func NewClient(
	conn net.Conn,
	forwarder *harvester.Forwarder,
	splitFunc bufio.SplitFunc,
	maxReadMessage uint64,
	timeout time.Duration,
) *Client {
	client := &Client{
		conn:           conn,
		forwarder:      forwarder,
		done:           make(chan struct{}),
		splitFunc:      splitFunc,
		maxReadMessage: maxReadMessage,
		timeout:        timeout,
		metadata: common.MapStr{
			"hostnames":  remoteHosts(conn),
			"ip_address": conn.RemoteAddr().String(),
		},
	}

	return client
}

// Handle is reading message from the specified TCP socket.
func (c *Client) Handle() error {
	r := NewResetableLimitedReader(NewDeadlineReader(c.conn, c.timeout), c.maxReadMessage)
	buf := bufio.NewReader(r)
	scanner := bufio.NewScanner(buf)
	scanner.Split(c.splitFunc)

	for scanner.Scan() {
		err := scanner.Err()
		if err != nil {
			// we are forcing a close on the socket, lets ignore any error that could happen.
			select {
			case <-c.done:
				break
			default:
			}
			// This is a user defined limit and we should notify the user.
			if IsMaxReadBufferErr(err) {
				logp.Err("tcp client error: %s", err)
			}
			return errors.Wrap(err, "tcp client error")
		}
		r.Reset()
		c.forwarder.Send(c.createEvent(scanner.Text()))
	}
	return nil
}

// Close stops reading from the socket and close the connection.
func (c *Client) Close() {
	close(c.done)
	c.conn.Close()
}

func (c *Client) createEvent(rawString string) *util.Data {
	data := util.NewData()
	data.Event = beat.Event{
		Timestamp: time.Now(),
		Meta:      c.metadata,
		Fields: common.MapStr{
			"message": rawString,
		},
	}
	return data
}

// GetRemoteHosts take the IP address of the client and try to resolve the name, if it fails we
// fallback to the IP, IP can resolve to multiple hostname.
func remoteHosts(conn net.Conn) []string {
	ip := conn.RemoteAddr().String()
	idx := strings.Index(ip, ":")
	if idx == -1 {
		return []string{ip}
	}
	ip = ip[0:idx]
	hosts, err := net.LookupAddr(ip)
	if err != nil {
		hosts = []string{ip}
	}
	return hosts
}
