package dialchain

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

// DialerChain composes builders for multiple network layers, used to build
// the final transport.Dialer object based on the network layers.
// Each layer can hold individual configurations. Use 'Clone' to copy and replace/wrap
// layers at will.
// Once all Layers have been prepared, use Build to build a transport.Dialer that can
// used with any go library network packages relying on standard library based dialers.
//
// For Additional Layering capabilities, DialerChain implements the NetDialer interface.
type DialerChain struct {
	Net    NetDialer
	Layers []Layer
}

// NetDialer provides the most low-level network layer for setting up a network
// connection. NetDialer objects do not support wrapping any lower network layers.
type NetDialer func(common.MapStr) (transport.Dialer, error)

// Layer is a configured network layer, wrapping any lower-level network layers.
type Layer func(common.MapStr, transport.Dialer) (transport.Dialer, error)

// Clone create a shallow copy of c.
func (c *DialerChain) Clone() *DialerChain {
	d := &DialerChain{
		Net:    c.Net,
		Layers: make([]Layer, len(c.Layers)),
	}
	copy(d.Layers, c.Layers)
	return d
}

// Build create a new transport.Dialer for use with other networking libraries.
func (c *DialerChain) Build(event common.MapStr) (d transport.Dialer, err error) {
	d, err = c.Net.build(event)
	if err != nil {
		return
	}

	for _, layer := range c.Layers {
		if d, err = layer.build(event, d); err != nil {
			return nil, err
		}
	}
	return
}

// AddLayer adds another layer to the dialer chain.
// The layer being added is the new topmost network layer using the other
// already present layers on dial.
func (c *DialerChain) AddLayer(l Layer) {
	c.Layers = append(c.Layers, l)
}

// TestBuild tries to build the DialerChain and reports any error reported by
// one of the layers.
func (c *DialerChain) TestBuild() error {
	_, err := c.Build(common.MapStr{})
	return err
}

func (d NetDialer) build(event common.MapStr) (transport.Dialer, error) {
	return d(event)
}

func (l Layer) build(event common.MapStr, next transport.Dialer) (transport.Dialer, error) {
	return l(event, next)
}
