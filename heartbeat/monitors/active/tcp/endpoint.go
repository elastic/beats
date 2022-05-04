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
)

// endpoint configures a host with all port numbers to be monitored by a dialer
// based job.
type endpoint struct {
	Scheme   string
	Hostname string
	Ports    []uint16
}

// perPortURLs returns a list containing one URL per port
func (e endpoint) perPortURLs() (urls []*url.URL) {
	for _, port := range e.Ports {
		urls = append(urls, &url.URL{
			Scheme: e.Scheme,
			Host:   net.JoinHostPort(e.Hostname, strconv.Itoa(int(port))),
		})
	}

	return urls
}

// makeEndpoints creates a single endpoint struct for each host/port permutation.
// Set `defaultScheme` to choose which scheme is used if not explicit in the host config.
func makeEndpoints(hosts []string, ports []uint16, defaultScheme string) (endpoints []endpoint, err error) {
	for _, h := range hosts {
		u, err := url.Parse(h)

		// If h is just a bare hostname like 'localhost' it will be parsed as the URL path, and host will
		// be blank
		var ep endpoint
		if err == nil && u.Host != "" {
			ep, err = makeURLEndpoint(u, ports)
			if err != nil {
				return nil, err
			}
		} else {
			u := &url.URL{Scheme: defaultScheme, Host: h}
			ep, err = makeURLEndpoint(u, ports)
			if err != nil {
				return nil, err
			}
		}
		endpoints = append(endpoints, ep)
	}
	return endpoints, nil
}

func makeURLEndpoint(u *url.URL, ports []uint16) (endpoint, error) {
	switch u.Scheme {
	case "tcp", "plain", "tls", "ssl":
	default:
		err := fmt.Errorf(
			"'%s' is not a supported connection scheme in '%s', supported schemes are tcp, plain, tls, and ssl",
			u.Scheme,
			u,
		)
		return endpoint{}, err
	}

	if u.Port() != "" {
		pUint, err := strconv.ParseUint(u.Port(), 10, 16)
		if err != nil {
			return endpoint{}, fmt.Errorf("no port(s) defined for TCP endpoint %s: %w", u, err)
		}
		ports = []uint16{uint16(pUint)}
	}

	if len(ports) == 0 {
		return endpoint{}, fmt.Errorf("host '%s' missing port number", u)
	}

	if u.Hostname() == "" || u.Hostname() == ":" {
		return endpoint{}, fmt.Errorf("could not parse tcp host '%s'", u)
	}

	return endpoint{
		Scheme:   u.Scheme,
		Hostname: u.Hostname(),
		Ports:    ports,
	}, nil
}
