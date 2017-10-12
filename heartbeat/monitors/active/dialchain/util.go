package dialchain

import (
	"net"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type timer struct {
	s, e time.Time
}

// IDLayer creates an empty placeholder layer.
func IDLayer() Layer {
	return _idLayer
}

var _idLayer = Layer(func(event common.MapStr, next transport.Dialer) (transport.Dialer, error) {
	return next, nil
})

// ConstAddrLayer introduces a network layer always passing a constant address
// to the underlying layer.
func ConstAddrLayer(address string) Layer {
	build := constAddr(address)

	return func(event common.MapStr, next transport.Dialer) (transport.Dialer, error) {
		return build(next), nil
	}
}

// MakeConstAddrLayer always passes the same address to the original Layer.
// This is useful if a lookup did return multiple IPs for the same hostname,
// but the IP use to connect shall be fixed.
func MakeConstAddrLayer(addr string, origLayer Layer) Layer {
	return withLayerDialer(origLayer, constAddr(addr))
}

// MakeConstAddrDialer always passes the same address to the original NetDialer.
// This is useful if a lookup did return multiple IPs for the same hostname,
// but the IP use to connect shall be fixed.
func MakeConstAddrDialer(addr string, origNet NetDialer) NetDialer {
	return withNetDialer(origNet, constAddr(addr))
}

func (t *timer) start()                  { t.s = time.Now() }
func (t *timer) stop()                   { t.e = time.Now() }
func (t *timer) duration() time.Duration { return t.e.Sub(t.s) }

// makeDialer aliases transport.DialerFunc
func makeDialer(fn func(network, address string) (net.Conn, error)) transport.Dialer {
	return transport.DialerFunc(fn)
}

// beforeDial will always call fn before executing the underlying dialer.
// The callback must return the original or a new address to be used with
// the dialer.
func beforeDial(dialer transport.Dialer, fn func(string) string) transport.Dialer {
	return makeDialer(func(network, address string) (net.Conn, error) {
		address = fn(address)
		return dialer.Dial(network, address)
	})
}

// afterDial will run fn after the dialer did successfully return a connection.
func afterDial(dialer transport.Dialer, fn func(net.Conn) (net.Conn, error)) transport.Dialer {
	return makeDialer(func(network, address string) (net.Conn, error) {
		conn, err := dialer.Dial(network, address)
		if err == nil {
			conn, err = fn(conn)
		}
		return conn, err
	})
}

func startTimerAfterDial(t *timer, dialer transport.Dialer) transport.Dialer {
	return afterDial(dialer, func(c net.Conn) (net.Conn, error) {
		t.start()
		return c, nil
	})
}

func constAddr(addr string) func(transport.Dialer) transport.Dialer {
	return func(dialer transport.Dialer) transport.Dialer {
		return beforeDial(dialer, func(_ string) string {
			return addr
		})
	}
}

func withNetDialer(layer NetDialer, fn func(transport.Dialer) transport.Dialer) NetDialer {
	return func(event common.MapStr) (transport.Dialer, error) {
		origDialer, err := layer.build(event)
		if err != nil {
			return nil, err
		}
		return fn(origDialer), nil
	}
}

func withLayerDialer(layer Layer, fn func(transport.Dialer) transport.Dialer) Layer {
	return func(event common.MapStr, next transport.Dialer) (transport.Dialer, error) {
		origDialer, err := layer.build(event, next)
		if err != nil {
			return nil, err
		}
		return fn(origDialer), nil
	}
}
