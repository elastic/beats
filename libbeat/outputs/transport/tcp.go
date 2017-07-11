package transport

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/testing"
)

func NetDialer(timeout time.Duration) Dialer {
	return TestNetDialer(testing.NullDriver, timeout)
}

func TestNetDialer(d testing.Driver, timeout time.Duration) Dialer {
	return DialerFunc(func(network, address string) (net.Conn, error) {
		switch network {
		case "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6":
		default:
			d.Fatal("network type", fmt.Errorf("unsupported network type %v", network))
			return nil, fmt.Errorf("unsupported network type %v", network)
		}

		host, port, err := net.SplitHostPort(address)
		d.Fatal("parse host", err)
		if err != nil {
			return nil, err
		}

		addresses, err := net.LookupHost(host)
		d.Fatal("dns lookup", err)
		d.Info("addresses", strings.Join(addresses, ", "))
		if err != nil {
			logp.Warn(`DNS lookup failure "%s": %v`, host, err)
			return nil, err
		}

		// dial via host IP by randomized iteration of known IPs
		dialer := &net.Dialer{Timeout: timeout}
		return DialWith(dialer, network, host, addresses, port)
	})
}
