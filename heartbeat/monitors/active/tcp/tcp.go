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
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/v7/heartbeat/eventext"
	"github.com/elastic/beats/v7/heartbeat/look"
	"github.com/elastic/beats/v7/heartbeat/reason"

	"github.com/elastic/beats/v7/heartbeat/monitors"
	"github.com/elastic/beats/v7/heartbeat/monitors/active/dialchain"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/heartbeat/monitors/wrappers"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func init() {
	monitors.RegisterActive("tcp", create)
}

var debugf = logp.MakeDebug("tcp")

func create(
	name string,
	cfg *common.Config,
) (jobs []jobs.Job, endpoints int, err error) {
	jc, err := MakeJobFactory(cfg)
	if err != nil {
		return nil, 0, err
	}

	jobs, err = jc.makeJobs()
	if err != nil {
		return nil, 0, err
	}

	return jobs, len(jc.endpoints), nil
}

type jobFactory struct {
	config        Config
	tlsConfig     *tlscommon.TLSConfig
	defaultScheme string
	endpoints     []Endpoint
	dataCheck     DataCheck
}

func MakeJobFactory(commonCfg *common.Config) (jf *jobFactory, err error) {
	jf = &jobFactory{config: DefaultConfig}
	err = jf.loadConfig(commonCfg)
	if err != nil {
		return nil, err
	}

	return jf, nil
}

func (jf *jobFactory) loadConfig(commonCfg *common.Config) (err error) {
	if err := commonCfg.Unpack(&jf.config); err != nil {
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

	jf.endpoints, err = makeEndpoints(&jf.config, jf.defaultScheme)
	if err != nil {
		return err
	}

	jf.dataCheck = makeDataCheck(&jf.config)

	return nil
}

func (jf *jobFactory) makeJobs() ([]jobs.Job, error) {
	var jobs []jobs.Job
	for _, endpoint := range jf.endpoints {
		for _, url := range endpoint.perPortURLs() {
			endpointJob, err := jf.makeEndpointJobFor(url)
			if err != nil {
				return nil, err
			}
			jobs = append(jobs, wrappers.WithURLField(url, endpointJob))
		}

	}

	return jobs, nil
}

func (jf *jobFactory) makeEndpointJobFor(endpointURL *url.URL) (jobs.Job, error) {
	// Check if SOCKS5 is configured, with relying on the socks5 proxy
	// in resolving the actual IP.
	// Create one job for every port number configured.
	if jf.config.Socks5.URL != "" && !jf.config.Socks5.LocalResolve {
		return wrappers.WithURLField(endpointURL,
			jobs.MakeSimpleJob(func(event *beat.Event) error {
				hostPort := net.JoinHostPort(endpointURL.Hostname(), endpointURL.Port())
				return jf.dial(event, hostPort, endpointURL)
			})), nil
	}

	// Create job that first resolves one or multiple IP (depending on
	// config.Mode) in order to create one continuation Task per IP.
	settings := monitors.MakeHostJobSettings(endpointURL.Hostname(), jf.config.Mode)

	job, err := monitors.MakeByHostJob(settings,
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

func (jf *jobFactory) dial(event *beat.Event, dialAddr string, canonicalURL *url.URL) error {
	dc := &dialchain.DialerChain{
		Net: dialchain.MakeConstAddrDialer(dialAddr, dialchain.TCPDialer(jf.config.Timeout)),
	}
	if jf.config.Socks5.URL != "" {
		dc.AddLayer(dialchain.SOCKS5Layer(&jf.config.Socks5))
	}

	isTLS := true
	if canonicalURL.Scheme == "tcp" || canonicalURL.Scheme == "plain" {
		isTLS = false
	}
	if isTLS {
		dc.AddLayer(dialchain.TLSLayer(jf.tlsConfig, jf.config.Timeout))
		dc.AddLayer(dialchain.ConstAddrLayer(canonicalURL.Host))
	}

	dialer, err := dc.Build(event)
	if err != nil {
		return err
	}

	return jf.pingAddr(event, dialer, dialAddr)
}

func (jf *jobFactory) pingAddr(
	event *beat.Event,
	dialer transport.Dialer,
	addr string,
) error {
	start := time.Now()
	deadline := start.Add(jf.config.Timeout)

	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		debugf("dial failed with: %v", err)
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

func makeEndpoints(config *Config, defaultScheme string) (endpoints []Endpoint, err error) {
	for _, h := range config.Hosts {
		scheme := defaultScheme
		host := ""
		u, err := url.Parse(h)

		if err != nil || u.Host == "" {
			host = h
		} else {
			scheme = u.Scheme
			host = u.Host
		}
		debugf("Add tcp endpoint '%v://%v'.", scheme, host)

		switch scheme {
		case "tcp", "plain", "tls", "ssl":
		default:
			err := fmt.Errorf("'%v' is not a supported connection scheme in '%v'", scheme, h)
			return nil, err
		}

		pair := strings.SplitN(host, ":", 2)
		ports := config.Ports
		if len(pair) == 2 {
			port, err := strconv.ParseUint(pair[1], 10, 16)
			if err != nil {
				return nil, fmt.Errorf("'%v' is no valid port number in '%v'", pair[1], h)
			}

			ports = []uint16{uint16(port)}
			host = pair[0]
		} else if len(config.Ports) == 0 {
			return nil, fmt.Errorf("host '%v' missing port number", h)
		}

		endpoints = append(endpoints, Endpoint{
			Scheme:   scheme,
			Hostname: host,
			Ports:    ports,
		})
	}
	return endpoints, nil
}
