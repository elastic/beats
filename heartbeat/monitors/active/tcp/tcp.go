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
	"net/url"
	"strconv"
	"strings"

	"github.com/elastic/beats/heartbeat/monitors"
	"github.com/elastic/beats/heartbeat/monitors/active/dialchain"
	"github.com/elastic/beats/heartbeat/monitors/jobs"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"
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
	config := DefaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, 0, err
	}

	tls, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, 0, err
	}

	defaultScheme := "tcp"
	if tls != nil {
		defaultScheme = "ssl"
	}

	schemeHosts, err := collectHosts(&config, defaultScheme)
	if err != nil {
		return nil, 0, err
	}

	timeout := config.Timeout
	validator := makeValidateConn(&config)

	for scheme, eps := range schemeHosts {
		schemeTLS := tls
		if scheme == "tcp" || scheme == "plain" {
			schemeTLS = nil
		}

		db, err := dialchain.NewBuilder(dialchain.BuilderSettings{
			Timeout: timeout,
			Socks5:  config.Socks5,
			TLS:     schemeTLS,
		})
		if err != nil {
			return nil, 0, err
		}

		epJobs, err := dialchain.MakeDialerJobs(db, scheme, eps, config.Mode,
			func(event *beat.Event, dialer transport.Dialer, addr string) error {
				return pingHost(event, dialer, addr, timeout, validator)
			})
		if err != nil {
			return nil, 0, err
		}

		jobs = append(jobs, epJobs...)
	}

	numHosts := 0
	for _, hosts := range schemeHosts {
		numHosts += len(hosts)
	}

	return jobs, numHosts, nil
}

func collectHosts(config *Config, defaultScheme string) (map[string][]dialchain.Endpoint, error) {
	endpoints := map[string][]dialchain.Endpoint{}
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

		endpoints[scheme] = append(endpoints[scheme], dialchain.Endpoint{
			Host:  host,
			Ports: ports,
		})
	}
	return endpoints, nil
}
