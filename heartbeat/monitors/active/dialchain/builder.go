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

package dialchain

import (
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/heartbeat/monitors/jobs"
	"github.com/elastic/beats/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

// Builder maintains a DialerChain for building dialers and dialer based
// monitoring jobs.
// The builder ensures a constant address is being used, for any host
// configured. This ensures the upper network layers (e.g. TLS) correctly see
// and process the original hostname.
type Builder struct {
	template         *DialerChain
	addrIndex        int
	resolveViaSocks5 bool
}

// BuilderSettings configures the layers of the dialer chain to be constructed
// by a Builder.
type BuilderSettings struct {
	Timeout time.Duration
	Socks5  transport.ProxyConfig
	TLS     *transport.TLSConfig
}

// Endpoint configures a host with all port numbers to be monitored by a dialer
// based job.
type Endpoint struct {
	Host  string
	Ports []uint16
}

// NewBuilder creates a new Builder for constructing dialers.
func NewBuilder(settings BuilderSettings) (*Builder, error) {
	d := &DialerChain{
		Net: netDialer(settings.Timeout),
	}
	resolveViaSocks5 := false
	withProxy := settings.Socks5.URL != ""
	if withProxy {
		d.AddLayer(SOCKS5Layer(&settings.Socks5))
		resolveViaSocks5 = !settings.Socks5.LocalResolve
	}

	// insert empty placeholder, so address can be replaced in dialer chain
	// by replacing this placeholder dialer
	idx := len(d.Layers)
	d.AddLayer(IDLayer())

	// add tls layer doing the TLS handshake based on the original address
	if tls := settings.TLS; tls != nil {
		d.AddLayer(TLSLayer(tls, settings.Timeout))
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
func (b *Builder) AddLayer(l Layer) {
	b.template.AddLayer(l)
}

// Build create a new dialer, that will always use the constant address, no matter
// which address is used to connect using the dialer.
// The dialer chain will add per layer information to the given event.
func (b *Builder) Build(addr string, event *beat.Event) (transport.Dialer, error) {
	// clone template, as multiple instance of a dialer can exist at the same time
	dchain := b.template.Clone()

	// fix the final dialers TCP-level address
	dchain.Layers[b.addrIndex] = ConstAddrLayer(addr)

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

// MakeDialerJobs creates a set of monitoring jobs. The jobs behavior depends
// on the builder, endpoint and mode configurations, normally set by user
// configuration.  The task to execute the actual 'ping' receives the dialer
// and the address pair (<hostname>:<port>), required to be used, to ping the
// correctly resolved endpoint.
func MakeDialerJobs(
	b *Builder,
	scheme string,
	endpoints []Endpoint,
	mode monitors.IPSettings,
	fn func(event *beat.Event, dialer transport.Dialer, addr string) error,
) ([]jobs.Job, error) {
	var jobs []jobs.Job
	for _, endpoint := range endpoints {
		for _, port := range endpoint.Ports {
			endpointURL, err := url.Parse(fmt.Sprintf("%s://%s:%d", scheme, endpoint.Host, port))
			if err != nil {
				return nil, err
			}
			endpointJob, err := makeEndpointJob(b, endpointURL, mode, fn)
			if err != nil {
				return nil, err
			}
			jobs = append(jobs, wrappers.WithURLField(endpointURL, endpointJob))
		}

	}

	return jobs, nil
}

func makeEndpointJob(
	b *Builder,
	endpointURL *url.URL,
	mode monitors.IPSettings,
	fn func(*beat.Event, transport.Dialer, string) error,
) (jobs.Job, error) {

	// Check if SOCKS5 is configured, with relying on the socks5 proxy
	// in resolving the actual IP.
	// Create one job for every port number configured.
	if b.resolveViaSocks5 {
		return wrappers.WithURLField(endpointURL,
			jobs.MakeSimpleJob(func(event *beat.Event) error {
				hostPort := net.JoinHostPort(endpointURL.Hostname(), endpointURL.Port())
				return b.Run(event, hostPort, func(event *beat.Event, dialer transport.Dialer) error {
					return fn(event, dialer, hostPort)
				})
			})), nil
	}

	// Create job that first resolves one or multiple IP (depending on
	// config.Mode) in order to create one continuation Task per IP.
	settings := monitors.MakeHostJobSettings(endpointURL.Hostname(), mode)

	job, err := monitors.MakeByHostJob(settings,
		monitors.MakePingIPFactory(
			func(event *beat.Event, ip *net.IPAddr) error {
				// use address from resolved IP
				ipPort := net.JoinHostPort(ip.String(), endpointURL.Port())
				cb := func(event *beat.Event, dialer transport.Dialer) error {
					return fn(event, dialer, ipPort)
				}
				err := b.Run(event, ipPort, cb)
				return err
			}))
	if err != nil {
		return nil, err
	}
	return job, nil
}
