package transport

import (
	"fmt"
	"net"
	"time"

	"github.com/elastic/beats/libbeat/logp"
)

func NetDialer(timeout time.Duration) Dialer {
	return DialerFunc(func(network, address string) (net.Conn, error) {
		switch network {
		case "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6":
		default:
			return nil, fmt.Errorf("unsupported network type %v", network)
		}

		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}

		addresses, err := net.LookupHost(host)
		if err != nil {
			logp.Warn(`DNS lookup failure "%s": %v`, host, err)
			return nil, err
		}

		// dial via host IP by randomized iteration of known IPs
		dialer := &net.Dialer{Timeout: timeout}
		return dialWith(dialer, network, host, addresses, port)
	})
}
