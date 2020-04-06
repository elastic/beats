package tcp

import (
	"net"
	"net/url"
	"strconv"
)

// Endpoint configures a host with all port numbers to be monitored by a dialer
// based job.
type Endpoint struct {
	Scheme   string
	Hostname string
	Ports    []uint16
}

// perPortURLs returns a list containing one URL per port
func (e Endpoint) perPortURLs() (urls []*url.URL) {
	for _, port := range e.Ports {
		urls = append(urls, &url.URL{
			Scheme: e.Scheme,
			Host:   net.JoinHostPort(e.Hostname, strconv.Itoa(int(port))),
		})
	}

	return urls
}
