package transport

import "net"

func ConnWrapper(d Dialer, w func(net.Conn) net.Conn) Dialer {
	return DialerFunc(func(network, addr string) (net.Conn, error) {
		c, err := d.Dial(network, addr)
		if err != nil {
			return nil, err
		}
		return w(c), nil
	})
}
