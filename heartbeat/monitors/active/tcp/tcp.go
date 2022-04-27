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
	"crypto/x509"
	"net"
	"net/url"
	"time"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/look"
	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/monitors/active/dialchain"
	"github.com/elastic/beats/v7/heartbeat/monitors/active/dialchain/tlsmeta"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/v7/heartbeat/reason"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/logp"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func init() {
	plugin.Register("tcp", create, "synthetics/tcp")
}

var debugf = logp.MakeDebug("tcp")

func create(
	name string,
	cfg *conf.C,
) (p plugin.Plugin, err error) {
	return createWithResolver(cfg, monitors.NewStdResolver())
}

// Custom resolver is useful for tests against hostnames locally where we don't want to depend on any
// hostnames existing in test environments
func createWithResolver(
	cfg *conf.C,
	resolver monitors.Resolver,
) (p plugin.Plugin, err error) {
	jc, err := newJobFactory(cfg, resolver)
	if err != nil {
		return plugin.Plugin{}, err
	}

	js, err := jc.makeJobs()
	if err != nil {
		return plugin.Plugin{}, err
	}

	return plugin.Plugin{Jobs: js, Endpoints: len(jc.endpoints)}, nil
}

// jobFactory is where most of the logic here lives. It provides a common context around
// the complex logic of executing a TCP check.
type jobFactory struct {
	config        config
	tlsConfig     *tlscommon.TLSConfig
	defaultScheme string
	endpoints     []endpoint
	dataCheck     dataCheck
	resolver      monitors.Resolver
}

func newJobFactory(commonCfg *conf.C, resolver monitors.Resolver) (*jobFactory, error) {
	jf := &jobFactory{config: defaultConfig(), resolver: resolver}
	err := jf.loadConfig(commonCfg)
	if err != nil {
		return nil, err
	}

	return jf, nil
}

// loadConfig parses the YAML config and populates the jobFactory fields.
func (jf *jobFactory) loadConfig(commonCfg *conf.C) error {
	var err error
	if err = commonCfg.Unpack(&jf.config); err != nil {
		return err
	}

	jf.tlsConfig, err = tlscommon.LoadTLSConfig(jf.config.TLS)
	if err != nil {
		return err
	}

	jf.defaultScheme = "tcp"
	if jf.tlsConfig != nil {
		jf.defaultScheme = "ssl"
	}

	jf.endpoints, err = makeEndpoints(jf.config.Hosts, jf.config.Ports, jf.defaultScheme)
	if err != nil {
		return err
	}

	jf.dataCheck = makeDataCheck(&jf.config)

	return nil
}

// makeJobs returns the actual schedulable jobs for this monitor.
func (jf *jobFactory) makeJobs() ([]jobs.Job, error) {
	var jobs []jobs.Job
	for _, endpoint := range jf.endpoints {
		for _, url := range endpoint.perPortURLs() {
			endpointJob, err := jf.makeEndpointJob(url)
			if err != nil {
				return nil, err
			}
			jobs = append(jobs, wrappers.WithURLField(url, endpointJob))
		}

	}

	return jobs, nil
}

// makeEndpointJob makes a job for a single check of a single scheme/host/port combo.
func (jf *jobFactory) makeEndpointJob(endpointURL *url.URL) (jobs.Job, error) {
	// Check if SOCKS5 is configured, with relying on the socks5 proxy
	// in resolving the actual IP.
	// Create one job for every port number configured.
	if jf.config.Socks5.URL != "" && !jf.config.Socks5.LocalResolve {
		jf.makeSocksLookupEndpointJob(endpointURL)
	}

	return jf.makeDirectEndpointJob(endpointURL)
}

// makeDirectEndpointJob makes jobs that directly lookup the IP of the endpoints, as opposed to using
// a Socks5 proxy.
func (jf *jobFactory) makeDirectEndpointJob(endpointURL *url.URL) (jobs.Job, error) {
	// Create job that first resolves one or multiple IPs (depending on
	// config.Mode) in order to create one continuation Task per IP.
	job, err := monitors.MakeByHostJob(
		endpointURL.Hostname(),
		jf.config.Mode,
		jf.resolver,
		monitors.MakePingIPFactory(
			func(event *beat.Event, ip *net.IPAddr) error {
				// use address from resolved IP
				ipPort := net.JoinHostPort(ip.String(), endpointURL.Port())

				return jf.dial(event, ipPort, endpointURL)
			}))
	if err != nil {
		return nil, err
	}
	return job, nil
}

// makeSocksLookupEndpointJob makes jobs that use a Socks5 proxy to perform DNS lookups
func (jf *jobFactory) makeSocksLookupEndpointJob(endpointURL *url.URL) (jobs.Job, error) {
	return wrappers.WithURLField(endpointURL,
		jobs.MakeSimpleJob(func(event *beat.Event) error {
			hostPort := net.JoinHostPort(endpointURL.Hostname(), endpointURL.Port())
			return jf.dial(event, hostPort, endpointURL)
		})), nil
}

// dial builds a dialer and executes the network request.
// dialAddr is the host:port that the dialer will connect to, and where an explicit IP should go to.
// canonicalURL is the URL used to determine if TLS is used via the scheme of the URL, and
// also which hostname should be passed to the TLS implementation for validation of the server cert.
func (jf *jobFactory) dial(event *beat.Event, dialAddr string, canonicalURL *url.URL) error {
	// First, create a plain dialer that can connect directly to either hostnames or IPs
	dc := &dialchain.DialerChain{
		Net: dialchain.CreateNetDialer(jf.config.Timeout),
	}

	// If Socks5 is configured make that the next layer, since everything needs to go through the proxy first.
	if jf.config.Socks5.URL != "" {
		dc.AddLayer(dialchain.SOCKS5Layer(&jf.config.Socks5))
	}

	// Now add the IP or Hostname of the server we want to connect to.
	// Usually this is the IP we've resolved in a prior step.
	// If we're using a proxy with host lookup enabled the dialAddr should be the
	// hostname we want the server to resolve for us.
	dc.AddLayer(dialchain.ConstAddrLayer(dialAddr))

	// If we're using TLS we need to add a fake layer so that the TLS layer knows the hostname we're connecting to
	// So, the canonical URL is fixed via a ConstAddrLayer to override the TLS layer's x509 logic so it doesn't
	// try and directly match the IP from the prior ConstAddrLayer to the cert.
	if canonicalURL.Scheme != "tcp" && canonicalURL.Scheme != "plain" {
		dc.AddLayer(dialchain.TLSLayer(jf.tlsConfig, jf.config.Timeout))
		dc.AddLayer(dialchain.ConstAddrLayer(canonicalURL.Host))
	}

	dialer, err := dc.Build(event)
	if err != nil {
		return err
	}

	return jf.execDialer(event, dialer, dialAddr)
}

// exec dialer executes a network request against the given dialer.
func (jf *jobFactory) execDialer(
	event *beat.Event,
	dialer transport.Dialer,
	addr string,
) error {
	start := time.Now()
	deadline := start.Add(jf.config.Timeout)

	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		debugf("dial failed with: %v", err)
		if certErr, ok := err.(x509.CertificateInvalidError); ok {
			tlsmeta.AddCertMetadata(event.Fields, []*x509.Certificate{certErr.Cert})
		}
		return reason.IOFailed(err)
	}
	defer conn.Close()
	if jf.dataCheck == nil {
		// no additional validation step => ping success
		return nil
	}

	if err := conn.SetDeadline(deadline); err != nil {
		debugf("setting connection deadline failed with: %v", err)
		return reason.IOFailed(err)
	}

	validateStart := time.Now()
	err = jf.dataCheck.Check(conn)
	if err != nil && err != errRecvMismatch {
		debugf("check failed with: %v", err)
		return reason.IOFailed(err)
	}

	end := time.Now()
	eventext.MergeEventFields(event, common.MapStr{
		"tcp": common.MapStr{
			"rtt": common.MapStr{
				"validate": look.RTT(end.Sub(validateStart)),
			},
		},
	})
	if err != nil {
		return reason.MakeValidateError(err)
	}

	return nil
}
