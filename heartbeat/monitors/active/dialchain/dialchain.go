package dialchain

import (
	"fmt"
	"net"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/transport"

	"github.com/elastic/beats/heartbeat/look"
)

type DialLayerCallback interface {
	Start()
	Done(err error)
}

type MeasureLayerRTTCB struct {
	Callback MeasureCallback
	start    time.Time
}

type MeasureCallback func(start, end time.Time)

type DialerChain struct {
	Net    NetDialer
	Layers []Layer
}

type NetDialer struct {
	Name   string
	Dialer transport.Dialer
}

type Layer struct {
	Name    string
	Builder func(transport.Dialer) (transport.Dialer, error)
}

func (c *DialerChain) Clone() *DialerChain {
	d := &DialerChain{
		Net:    c.Net,
		Layers: make([]Layer, len(c.Layers)),
	}
	copy(d.Layers, c.Layers)
	return d
}

func (c *DialerChain) BuildWith(makeCB func(string) DialLayerCallback) (d transport.Dialer, err error) {
	d = LayerCBDialer(makeCB(c.Net.Name), c.Net.Dialer)
	for _, layer := range c.Layers {
		if d, err = LayerDeltaCBDialer(makeCB(layer.Name), d, layer.Builder); err != nil {
			return nil, err
		}
	}
	return
}

func (c *DialerChain) BuildWithMeasures(event common.MapStr) (transport.Dialer, error) {
	return c.BuildWith(func(name string) DialLayerCallback {
		return measureEventRTT(event, name)
	})
}

func (c *DialerChain) Build() (d transport.Dialer, err error) {
	d = c.Net.Dialer
	for _, layer := range c.Layers {
		if d, err = layer.Builder(d); err != nil {
			return nil, err
		}
	}
	return
}

func (c *DialerChain) TestBuild() error {
	_, err := c.Build()
	return err
}

func (c *DialerChain) DialWithMeasurements(network, host string) (fields common.MapStr, conn net.Conn, err error) {
	var dialer transport.Dialer
	fields = common.MapStr{}
	if dialer, err = c.BuildWithMeasures(fields); err == nil {
		conn, err = dialer.Dial(network, host)
	}
	return
}

func (c *DialerChain) Dial(network, host string) (conn net.Conn, err error) {
	var dialer transport.Dialer
	if dialer, err = c.Build(); err == nil {
		return dialer.Dial(network, host)
	}
	return
}

func (c *DialerChain) AddLayer(l Layer) {
	c.Layers = append(c.Layers, l)
}

func measureEventRTT(event common.MapStr, name string) DialLayerCallback {
	return &MeasureLayerRTTCB{Callback: func(start, end time.Time) {
		event[name] = look.RTT(end.Sub(start))
	}}
}

func LayerCBDialer(cb DialLayerCallback, d transport.Dialer) transport.Dialer {
	return transport.DialerFunc(func(network, address string) (net.Conn, error) {
		cb.Start()
		c, err := d.Dial(network, address)
		cb.Done(err)
		return c, err
	})
}

func LayerDeltaCBDialer(
	cb DialLayerCallback,
	dialer transport.Dialer,
	layer func(transport.Dialer) (transport.Dialer, error),
) (transport.DialerFunc, error) {
	starter := transport.DialerFunc(func(network, address string) (net.Conn, error) {
		c, err := dialer.Dial(network, address)
		cb.Start()
		return c, err
	})

	layerInstance, err := layer(starter)
	if err != nil {
		return nil, err
	}

	return func(network, address string) (net.Conn, error) {
		c, err := layerInstance.Dial(network, address)
		cb.Done(err)
		return c, err
	}, nil
}

func ConstAddrDialer(name, addr string, to time.Duration) NetDialer {
	return NetDialer{name, transport.DialerFunc(func(network, _ string) (net.Conn, error) {
		switch network {
		case "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6":
		default:
			return nil, fmt.Errorf("unsupported network type %v", network)
		}

		dialer := &net.Dialer{Timeout: to}
		return dialer.Dial(network, addr)
	})}
}

func ConstAddrLayer(addr string, l Layer) Layer {
	return Layer{l.Name, func(d transport.Dialer) (transport.Dialer, error) {
		forward, err := l.Builder(d)
		if err != nil {
			return nil, err
		}

		return transport.DialerFunc(func(network, _ string) (net.Conn, error) {
			return forward.Dial(network, addr)
		}), nil
	}}
}

func TCPDialer(name string, to time.Duration) NetDialer {
	return NetDialer{name, transport.NetDialer(to)}
}

func UDPDialer(name string, to time.Duration) NetDialer {
	return NetDialer{name, transport.NetDialer(to)}
}

func SOCKS5Layer(name string, config *transport.ProxyConfig) Layer {
	return Layer{name, func(d transport.Dialer) (transport.Dialer, error) {
		return transport.ProxyDialer(config, d)
	}}
}

func TLSLayer(name string, config *transport.TLSConfig, timeout time.Duration) Layer {
	return Layer{name, func(d transport.Dialer) (transport.Dialer, error) {
		return transport.TLSDialer(d, config, timeout)
	}}
}

func (cb *MeasureLayerRTTCB) Start()       { cb.start = time.Now() }
func (cb *MeasureLayerRTTCB) Done(_ error) { cb.Callback(cb.start, time.Now()) }
