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
	"errors"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/rcrowley/go-metrics"

	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

const etcResolvConf = "/etc/resolv.conf"

// result represents a DNS lookup result.
type result struct {
	Data []string // Hostname.
	TTL  uint32   // Time to live in seconds.
}

// resolver performs result record lookups.
type resolver interface {
	Lookup(q string, qt queryType) (*result, error)
}

// miekgResolver is a resolver that is implemented using github.com/miekg/dns
// to send requests to DNS servers. It does not use the Go resolver.
type miekgResolver struct {
	client  *dns.Client
	servers []string

	registry     *monitoring.Registry
	nsStatsMutex sync.RWMutex
	nsStats      map[string]*nameserverStats
}

type nameserverStats struct {
	success         *monitoring.Int // Number of responses from server.
	failure         *monitoring.Int // Number of failures (e.g. I/O timeout) (not NXDOMAIN).
	requestDuration metrics.Sample  // Histogram of response times.
}

// newMiekgResolver returns a new miekgResolver. It returns an error if no
// nameserver are given and none can be read from /etc/resolv.conf.
func newMiekgResolver(reg *monitoring.Registry, timeout time.Duration, transport string, servers ...string) (*miekgResolver, error) {
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
		timeout = defaultConfig().Timeout
	}

	var clientTransferType string
	switch transport {
	case "tls":
		clientTransferType = "tcp-tls"
	default:
		clientTransferType = "udp"
	}

	return &miekgResolver{
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

// Lookup performs a DNS query.
func (res *miekgResolver) Lookup(q string, qt queryType) (*result, error) {
	if len(res.servers) == 0 {
		return nil, errors.New("no dns servers configured")
	}

	// Create DNS request.
	m := new(dns.Msg)
	switch qt {
	case typePTR:
		arpa, err := dns.ReverseAddr(q)
		if err != nil {
			return nil, err
		}
		m.SetQuestion(arpa, dns.TypePTR)
	case typeA, typeAAAA, typeTXT:
		m.SetQuestion(dns.Fqdn(q), uint16(qt))
	}
	m.RecursionDesired = true

	// Try the nameservers until we get a response.
	var nameserverErr error
	for _, server := range res.servers {
		stats := res.getOrCreateNameserverStats(server)

		r, rtt, err := res.client.Exchange(m, server)
		if err != nil {
			// Try next server if any. Otherwise, return nameserverErr.
			nameserverErr = err
			stats.failure.Inc()
			continue
		}

		// We got a response.
		stats.success.Inc()
		stats.requestDuration.Update(int64(rtt))
		if r.Rcode != dns.RcodeSuccess {
			name, found := dns.RcodeToString[r.Rcode]
			if !found {
				name = "response code " + strconv.Itoa(r.Rcode)
			}
			return nil, &dnsError{"nameserver " + server + " returned " + name}
		}

		var rtn result
		rtn.TTL = math.MaxUint32
		for _, a := range r.Answer {
			// Ignore records that don't match the query type.
			if a.Header().Rrtype != uint16(qt) {
				continue
			}

			switch rr := a.(type) {
			case *dns.PTR:
				return &result{
					Data: []string{strings.TrimSuffix(rr.Ptr, ".")},
					TTL:  rr.Hdr.Ttl,
				}, nil
			case *dns.A:
				rtn.Data = append(rtn.Data, rr.A.String())
				rtn.TTL = min(rtn.TTL, rr.Hdr.Ttl)
			case *dns.AAAA:
				rtn.Data = append(rtn.Data, rr.AAAA.String())
				rtn.TTL = min(rtn.TTL, rr.Hdr.Ttl)
			case *dns.TXT:
				rtn.Data = append(rtn.Data, rr.Txt...)
				rtn.TTL = min(rtn.TTL, rr.Hdr.Ttl)
			}
		}

		if len(rtn.Data) == 0 {
			return nil, &dnsError{"no " + qt.String() + " resource records were found in the response"}
		}

		return &rtn, nil
	}

	if nameserverErr != nil {
		return nil, nameserverErr
	}

	// This should never get here.
	panic("dns resolver Lookup() should have returned a response.")
}

func (res *miekgResolver) getOrCreateNameserverStats(ns string) *nameserverStats {
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
		success:         monitoring.NewInt(reg, "success"),
		failure:         monitoring.NewInt(reg, "failure"),
		requestDuration: metrics.NewUniformSample(1028),
	}

	//nolint:errcheck // Register should never fail because this is a new empty registry.
	adapter.NewGoMetrics(reg, "request_duration", adapter.Accept).
		Register("histogram", metrics.NewHistogram(stats.requestDuration))
	res.nsStats[ns] = stats

	return stats
}

func min(a, b uint32) uint32 {
	if a < b {
		return a
	}
	return b
}
