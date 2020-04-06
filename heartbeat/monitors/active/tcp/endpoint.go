package tcp

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
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
		if len(pair) == 2 {
			port, err := strconv.ParseUint(pair[1], 10, 16)
			if err != nil {
				return nil, fmt.Errorf("'%v' is no valid port number in '%v'", pair[1], h)
			}

			ports = []uint16{uint16(port)}
			host = pair[0]
		} else if len(ports) == 0 {
			return nil, fmt.Errorf("host '%v' missing port number", h)
		}

		endpoints = append(endpoints, endpoint{
			Scheme:   scheme,
			Hostname: host,
			Ports:    ports,
		})
	}
	return endpoints, nil
}
