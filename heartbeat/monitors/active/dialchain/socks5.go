package dialchain

import (
	"net"

	"github.com/elastic/beats/heartbeat/look"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

// SOCKS5Layer configures a SOCKS5 proxy layer in a DialerChain.
//
// The layer will update the active event with:
//
//  {
//    "socks5": {
//        "rtt": { "connect": { "us": ... }}
//    }
//  }
func SOCKS5Layer(config *transport.ProxyConfig) Layer {
	return func(event common.MapStr, next transport.Dialer) (transport.Dialer, error) {
		var timer timer

		dialer, err := transport.ProxyDialer(config, startTimerAfterDial(&timer, next))
		if err != nil {
			return nil, err
		}

		return afterDial(dialer, func(conn net.Conn) (net.Conn, error) {
			// TODO: extract connection parameter from connection object?
			// TODO: add proxy url to event?

			timer.stop()
			event.Put("socks5.rtt.connect", look.RTT(timer.duration()))
			return conn, nil
		}), nil
	}
}
