package v2

import "net"

// SyncClient synchronously publishes events to lumberjack endpoint waiting for
// ACK before allowing another send request. The client is not thread-safe.
type SyncClient struct {
	cl *Client
}

func NewSyncClientWith(c *Client) (*SyncClient, error) {
	return &SyncClient{c}, nil
}

func NewSyncClientWithConn(c net.Conn, opts ...Option) (*SyncClient, error) {
	cl, err := NewWithConn(c, opts...)
	if err != nil {
		return nil, err
	}
	return NewSyncClientWith(cl)
}

func SyncDial(address string, opts ...Option) (*SyncClient, error) {
	cl, err := Dial(address, opts...)
	if err != nil {
		return nil, err
	}
	return NewSyncClientWith(cl)
}

func SyncDialWith(
	dial func(network, address string) (net.Conn, error),
	address string,
	opts ...Option,
) (*SyncClient, error) {
	cl, err := DialWith(dial, address, opts...)
	if err != nil {
		return nil, err
	}
	return NewSyncClientWith(cl)
}

func (c *SyncClient) Close() error {
	return c.cl.Close()
}

func (c *SyncClient) Send(data []interface{}) (int, error) {
	if err := c.cl.Send(data); err != nil {
		return 0, err
	}

	seq, err := c.cl.AwaitACK(uint32(len(data)))
	return int(seq), err
}
