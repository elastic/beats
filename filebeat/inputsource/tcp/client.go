package tcp

import (
	"bufio"
	"net"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/filebeat/inputsource"
	"github.com/elastic/beats/libbeat/logp"
)

// Client is a remote client.
type client struct {
	conn           net.Conn
	log            *logp.Logger
	callback       inputsource.NetworkFunc
	done           chan struct{}
	metadata       inputsource.NetworkMetadata
	splitFunc      bufio.SplitFunc
	maxMessageSize uint64
	timeout        time.Duration
}

func newClient(
	conn net.Conn,
	log *logp.Logger,
	callback inputsource.NetworkFunc,
	splitFunc bufio.SplitFunc,
	maxReadMessage uint64,
	timeout time.Duration,
) *client {
	client := &client{
		conn:           conn,
		log:            log.With("address", conn.RemoteAddr()),
		callback:       callback,
		done:           make(chan struct{}),
		splitFunc:      splitFunc,
		maxMessageSize: maxReadMessage,
		timeout:        timeout,
		metadata: inputsource.NetworkMetadata{
			RemoteAddr: conn.RemoteAddr(),
		},
	}
	return client
}

func (c *client) handle() error {
	r := NewResetableLimitedReader(NewDeadlineReader(c.conn, c.timeout), c.maxMessageSize)
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
				c.log.Errorw("client errors", "error", err)
			}
			return errors.Wrap(err, "tcp client error")
		}
		r.Reset()
		c.callback(scanner.Bytes(), c.metadata)
	}
	return nil
}

func (c *client) close() {
	close(c.done)
	c.conn.Close()
}
