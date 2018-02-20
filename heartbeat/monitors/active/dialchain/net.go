package dialchain

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/elastic/beats/heartbeat/look"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

// TCPDialer creates a new NetDialer with constant event fields and default
// connection timeout.
// The fields parameter holds additional constants to be added to the final
// event structure.
//
// The dialer will update the active events with:
//
//  {
//    "tcp": {
//      "port": ...,
//      "rtt": { "connect": { "us": ... }}
//    }
//  }
func TCPDialer(to time.Duration) NetDialer {
	return netDialer(to)
}

// UDPDialer creates a new NetDialer with constant event fields and default
// connection timeout.
// The fields parameter holds additional constants to be added to the final
// event structure.
//
// The dialer will update the active events with:
//
//  {
//    "udp": {
//      "port": ...,
//      "rtt": { "connect": { "us": ... }}
//    }
//  }
func UDPDialer(to time.Duration) NetDialer {
	return netDialer(to)
}

func netDialer(timeout time.Duration) NetDialer {
	return func(event common.MapStr) (transport.Dialer, error) {
		return makeDialer(func(network, address string) (net.Conn, error) {
			namespace := ""

			switch network {
			case "tcp", "tcp4", "tcp6":
				namespace = "tcp"
			case "udp", "udp4", "udp6":
				namespace = "udp"
			default:
				return nil, fmt.Errorf("unsupported network type %v", network)
			}

			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}

			portNum, err := strconv.Atoi(port)
			if err != nil || portNum < 0 || portNum > (1<<16) {
				return nil, fmt.Errorf("invalid port number '%v' used", port)
			}
			event.DeepUpdate(common.MapStr{
				namespace: common.MapStr{
					"port": uint16(portNum),
				},
			})

			addresses, err := net.LookupHost(host)
			if err != nil {
				logp.Warn(`DNS lookup failure "%s": %v`, host, err)
				return nil, err
			}

			// dial via host IP by randomized iteration of known IPs
			dialer := &net.Dialer{Timeout: timeout}

			start := time.Now()
			conn, err := transport.DialWith(dialer, network, host, addresses, port)
			if err != nil {
				return nil, err
			}

			end := time.Now()
			event.DeepUpdate(common.MapStr{
				namespace: common.MapStr{
					"rtt": common.MapStr{
						"connect": look.RTT(end.Sub(start)),
					},
				},
			})

			return conn, nil
		}), nil
	}
}
