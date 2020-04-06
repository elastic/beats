// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package tcp

import (
	"time"

	"github.com/elastic/beats/v7/heartbeat/monitors/active/dialchain"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/transport"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
)

// Builder maintains a DialerChain for building dialers and dialer based
// monitoring jobs.
// The builder ensures a constant address is being used, for any host
// configured. This ensures the upper network layers (e.g. TLS) correctly see
// and process the original hostname.
type Builder struct {
	template         *dialchain.DialerChain
	addrIndex        int
	resolveViaSocks5 bool
}

// BuilderSettings configures the layers of the dialer chain to be constructed
// by a Builder.
type BuilderSettings struct {
	Timeout time.Duration
	Socks5  transport.ProxyConfig
	TLS     *tlscommon.TLSConfig
}

// Endpoint configures a host with all port numbers to be monitored by a dialer
// based job.
type Endpoint struct {
	Host  string
	Ports []uint16
}

// NewBuilder creates a new Builder for constructing dialers.
func NewBuilder(settings BuilderSettings) (*Builder, error) {
	d := &dialchain.DialerChain{
		Net: dialchain.CreateNetDialer(settings.Timeout),
	}
	resolveViaSocks5 := false
	withProxy := settings.Socks5.URL != ""
	if withProxy {
		d.AddLayer(dialchain.SOCKS5Layer(&settings.Socks5))
		resolveViaSocks5 = !settings.Socks5.LocalResolve
	}

	// insert empty placeholder, so address can be replaced in dialer chain
	// by replacing this placeholder dialer
	idx := len(d.Layers)
	d.AddLayer(dialchain.IDLayer())

	// add tls layer doing the TLS handshake based on the original address
	if tls := settings.TLS; tls != nil {
		d.AddLayer(dialchain.TLSLayer(tls, settings.Timeout))
	}

	// validate dialerchain
	if err := d.TestBuild(); err != nil {
		return nil, err
	}

	return &Builder{
		template:         d,
		addrIndex:        idx,
		resolveViaSocks5: resolveViaSocks5,
	}, nil
}

// AddLayer adds another custom network layer to the dialer chain.
func (b *Builder) AddLayer(l dialchain.Layer) {
	b.template.AddLayer(l)
}

// Build create a new dialer, that will always use the constant address, no matter
// which address is used to connect using the dialer.
// The dialer chain will add per layer information to the given event.
func (b *Builder) Build(addr string, event *beat.Event) (transport.Dialer, error) {
	// clone template, as multiple instance of a dialer can exist at the same time
	dchain := b.template.Clone()

	// fix the final dialers TCP-level address
	dchain.Layers[b.addrIndex] = dialchain.ConstAddrLayer(addr)

	// create dialer chain with event to add per network layer information
	d, err := dchain.Build(event)
	return d, err
}

// Run executes the given function with a new dialer instance.
func (b *Builder) Run(
	event *beat.Event,
	addr string,
	fn func(*beat.Event, transport.Dialer) error,
) error {
	dialer, err := b.Build(addr, event)
	if err != nil {
		return err
	}

	return fn(event, dialer)
}
