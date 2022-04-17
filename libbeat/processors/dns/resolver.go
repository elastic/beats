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

package dns

import (
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"github.com/rcrowley/go-metrics"

	"github.com/menderesk/beats/v7/libbeat/monitoring"
	"github.com/menderesk/beats/v7/libbeat/monitoring/adapter"
)

const etcResolvConf = "/etc/resolv.conf"

// PTR represents a DNS pointer record (IP to hostname).
type PTR struct {
	Host string // Hostname.
	TTL  uint32 // Time to live in seconds.
}

// PTRResolver performs PTR record lookups.
type PTRResolver interface {
	LookupPTR(ip string) (*PTR, error)
}

// MiekgResolver is a PTRResolver that is implemented using github.com/miekg/dns
// to send requests to DNS servers. It does not use the Go resolver.
type MiekgResolver struct {
	client  *dns.Client
	servers []string

	registry     *monitoring.Registry
	nsStatsMutex sync.RWMutex
	nsStats      map[string]*nameserverStats
}

type nameserverStats struct {
	success     *monitoring.Int // Number of responses from server.
	failure     *monitoring.Int // Number of failures (e.g. I/O timeout) (not NXDOMAIN).
	ptrResponse metrics.Sample  // Histogram of response times.
}

// NewMiekgResolver returns a new MiekgResolver. It returns an error if no
// nameserver are given and none can be read from /etc/resolv.conf.
func NewMiekgResolver(reg *monitoring.Registry, timeout time.Duration, transport string, servers ...string) (*MiekgResolver, error) {
	// Use /etc/resolv.conf if no nameservers are given. (Won't work for Windows).
	if len(servers) == 0 {
		config, err := dns.ClientConfigFromFile(etcResolvConf)
		if err != nil || len(config.Servers) == 0 {
			return nil, errors.New("no dns servers configured")
		}
		servers = config.Servers
	}

	// Add port if one was not specified.
	for i, s := range servers {
		if _, _, err := net.SplitHostPort(s); err != nil {
			var withPort string
			switch transport {
			case "tls":
				withPort = s + ":853"
			default:
				withPort = s + ":53"
			}

			if _, _, retryErr := net.SplitHostPort(withPort); retryErr == nil {
				servers[i] = withPort
				continue
			}
			return nil, err
		}
	}

	if timeout == 0 {
		timeout = defaultConfig.Timeout
	}

	var clientTransferType string
	switch transport {
	case "tls":
		clientTransferType = "tcp-tls"
	default:
		clientTransferType = "udp"
	}

	return &MiekgResolver{
		client: &dns.Client{
			Net:     clientTransferType,
			Timeout: timeout,
		},
		servers:  servers,
		registry: reg,
		nsStats:  map[string]*nameserverStats{},
	}, nil
}

// dnsError represents a failure response from the DNS server (like NXDOMAIN),
// but not a communication failure to the server. The response is cacheable.
type dnsError struct {
	err string
}

func (e *dnsError) Error() string {
	if e == nil {
		return "dns: <nil>"
	}
	return "dns: " + e.err
}

// LookupPTR performs a reverse lookup on the given IP address.
func (res *MiekgResolver) LookupPTR(ip string) (*PTR, error) {
	if len(res.servers) == 0 {
		return nil, errors.New("no dns servers configured")
	}

	// Create PTR (reverse) DNS request.
	m := new(dns.Msg)
	arpa, err := dns.ReverseAddr(ip)
	if err != nil {
		return nil, err
	}
	m.SetQuestion(arpa, dns.TypePTR)
	m.RecursionDesired = true

	// Try the nameservers until we get a response.
	var rtnErr error
	for _, server := range res.servers {
		stats := res.getOrCreateNameserverStats(server)

		r, rtt, err := res.client.Exchange(m, server)
		if err != nil {
			// Try next server if any. Otherwise return retErr.
			rtnErr = err
			stats.failure.Inc()
			continue
		}

		// We got a response.
		stats.success.Inc()
		stats.ptrResponse.Update(int64(rtt))
		if r.Rcode != dns.RcodeSuccess {
			name, found := dns.RcodeToString[r.Rcode]
			if !found {
				name = "response code " + strconv.Itoa(r.Rcode)
			}
			return nil, &dnsError{"nameserver " + server + " returned " + name}
		}

		for _, a := range r.Answer {
			if ptr, ok := a.(*dns.PTR); ok {
				return &PTR{
					Host: strings.TrimSuffix(ptr.Ptr, "."),
					TTL:  ptr.Hdr.Ttl,
				}, nil
			}
		}

		return nil, &dnsError{"no PTR record was found in the response"}
	}

	if rtnErr != nil {
		return nil, rtnErr
	}

	// This should never get here.
	panic("LookupPTR should have returned a response.")
}

func (res *MiekgResolver) getOrCreateNameserverStats(ns string) *nameserverStats {
	// Trim port.
	ns = ns[:strings.LastIndex(ns, ":")]

	// Check if stats already exist.
	res.nsStatsMutex.RLock()
	stats, found := res.nsStats[ns]
	if found {
		res.nsStatsMutex.RUnlock()
		return stats
	}
	res.nsStatsMutex.RUnlock()

	// Upgrade to a write lock and double-check.
	res.nsStatsMutex.Lock()
	defer res.nsStatsMutex.Unlock()
	stats, found = res.nsStats[ns]
	if found {
		return stats
	}

	// Create stats for the nameserver.
	reg := res.registry.NewRegistry(strings.Replace(ns, ".", "_", -1))
	stats = &nameserverStats{
		success:     monitoring.NewInt(reg, "success"),
		failure:     monitoring.NewInt(reg, "failure"),
		ptrResponse: metrics.NewUniformSample(1028),
	}
	adapter.NewGoMetrics(reg, "response.ptr", adapter.Accept).
		Register("histogram", metrics.NewHistogram(stats.ptrResponse))
	res.nsStats[ns] = stats

	return stats
}
