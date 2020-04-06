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

	"github.com/elastic/beats/v7/heartbeat/monitors"
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

type connURL struct {
	Scheme string
	Host   string
	Ports  []uint16
}

func create(
	name string,
	cfg *common.Config,
) (jobs []jobs.Job, endpoints int, err error) {
	tm, err := createTCPMonitor(cfg)
	if err != nil {
		return nil, 0, err
	}

	for scheme, eps := range tm.schemeHosts {
		/*
			db, err := NewBuilder(BuilderSettings{
				Timeout: tm.config.Timeout,
				Socks5:  tm.config.Socks5,
				TLS:     schemeTLS,
			})
			if err != nil {
				return nil, 0, err
			}
		*/

		epJobs, err := tm.MakeDialerJobsFor(scheme, eps,
			func(event *beat.Event, dialer transport.Dialer, addr string) error {
				return pingHost(event, dialer, addr, tm.config.Timeout, tm.dataCheck)
			})
		if err != nil {
			return nil, 0, err
		}

		jobs = append(jobs, epJobs...)
	}

	numHosts := 0
	for _, hosts := range tm.schemeHosts {
		numHosts += len(hosts)
	}

	return jobs, numHosts, nil
}

type tcpMonitor struct {
	config        Config
	tlsConfig     *tlscommon.TLSConfig
	defaultScheme string
	schemeHosts   map[string][]Endpoint
	dataCheck     DataCheck
}

func createTCPMonitor(commonCfg *common.Config) (tm *tcpMonitor, err error) {
	tm = &tcpMonitor{config: DefaultConfig}
	err = tm.loadConfig(commonCfg)
	if err != nil {
		return nil, err
	}

	return tm, nil
}

func (tm *tcpMonitor) MakeDialerJobsFor(
	scheme string,
	endpoints []Endpoint,
	fn func(event *beat.Event, dialer transport.Dialer, addr string) error,
) ([]jobs.Job, error) {
	var jobs []jobs.Job
	for _, endpoint := range endpoints {
		for _, port := range endpoint.Ports {
			endpointURL, err := url.Parse(fmt.Sprintf("%s://%s:%d", scheme, endpoint.Host, port))
			if err != nil {
				return nil, err
			}
			endpointJob, err := tm.makeEndpointJobFor(endpointURL, fn)
			if err != nil {
				return nil, err
			}
			jobs = append(jobs, wrappers.WithURLField(endpointURL, endpointJob))
		}

	}

	return jobs, nil
}

func (tm *tcpMonitor) makeEndpointJobFor(
	endpointURL *url.URL,
	fn func(*beat.Event, transport.Dialer, string) error,
) (jobs.Job, error) {

	// Check if SOCKS5 is configured, with relying on the socks5 proxy
	// in resolving the actual IP.
	// Create one job for every port number configured.
	/*
		if !tm.config.Socks5.LocalResolve {
			return wrappers.WithURLField(endpointURL,
				jobs.MakeSimpleJob(func(event *beat.Event) error {
					hostPort := net.JoinHostPort(endpointURL.Hostname(), endpointURL.Port())

					return b.Run(event, hostPort, func(event *beat.Event, dialer transport.Dialer) error {
						return fn(event, dialer, hostPort)
					})
				})), nil
		}
	*/

	// Create job that first resolves one or multiple IP (depending on
	// config.Mode) in order to create one continuation Task per IP.
	settings := monitors.MakeHostJobSettings(endpointURL.Hostname(), tm.config.Mode)

	job, err := monitors.MakeByHostJob(settings,
		monitors.MakePingIPFactory(
			func(event *beat.Event, ip *net.IPAddr) error {
				// use address from resolved IP
				ipPort := net.JoinHostPort(ip.String(), endpointURL.Port())
				cb := func(event *beat.Event, dialer transport.Dialer) error {
					return fn(event, dialer, ipPort)
				}

				schemeTLS := tm.tlsConfig
				if endpointURL.Scheme == "tcp" || endpointURL.Scheme == "plain" {
					schemeTLS = nil
				}

				db, err := NewBuilder(BuilderSettings{
					Timeout: tm.config.Timeout,
					TLS:     schemeTLS,
				})
				if err != nil {
					return err
				}

				return db.Run(event, ipPort, cb)
			}))
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (tm *tcpMonitor) loadConfig(commonCfg *common.Config) (err error) {
	if err := commonCfg.Unpack(&tm.config); err != nil {
		return err
	}

	tm.tlsConfig, err = tlscommon.LoadTLSConfig(tm.config.TLS)
	if err != nil {
		return err
	}

	tm.defaultScheme = "tcp"
	if tm.tlsConfig != nil {
		tm.defaultScheme = "ssl"
	}

	tm.schemeHosts, err = collectHosts(&tm.config, tm.defaultScheme)
	if err != nil {
		return err
	}

	tm.dataCheck = makeDataCheck(&tm.config)

	return nil
}

func collectHosts(config *Config, defaultScheme string) (map[string][]Endpoint, error) {
	endpoints := map[string][]Endpoint{}
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
			err := fmt.Errorf("'%v' is no supported connection scheme in '%v'", scheme, h)
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

		endpoints[scheme] = append(endpoints[scheme], Endpoint{
			Host:  host,
			Ports: ports,
		})
	}
	return endpoints, nil
}
