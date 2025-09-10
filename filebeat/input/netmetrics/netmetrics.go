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

// Package netmetrics provides different metricsets for capturing network-related metrics,
// such as UDP and TCP metrics through NewUDP and NewTCP, respectively.
package netmetrics

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/rcrowley/go-metrics"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

// addrs returns the linux /proc/net/tcp or /proc/net/udp addresses for the
// provided host address, addr. addr is a host:port address and host may be
// an IPv4 or IPv6 address, or an FQDN for the host. The returned slices
// contain the string representations of the address as they would appear in
// the /proc/net tables.
func addrs(addr string, log *logp.Logger) (addr4, addr6 []string, err error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get address for %s: could not split host and port: %w", addr, err)
	}
	ip, err := net.LookupIP(host)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get address for %s: %w", addr, err)
	}
	pn, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get port for %s: %w", addr, err)
	}
	addr4 = make([]string, 0, len(ip))
	addr6 = make([]string, 0, len(ip))
	for _, p := range ip {
		// Ensure the length of the net.IP is canonicalised to the standard
		// length for the format, as the net package may return IPv4 addresses
		// as the IPv6 form ::ffff:wwxxyyzz. So if we only compare len(p) to
		// the len constants all addresses may appear to be IPv6.
		switch {
		case len(p.To4()) == net.IPv4len:
			addr4 = append(addr4, ipV4(p, int(pn)))
		case len(p.To16()) == net.IPv6len:
			addr6 = append(addr6, ipV6(p, int(pn)))
		default:
			log.Warnf("unexpected addr length %d for %s", len(p), p)
		}
	}
	return addr4, addr6, nil
}

// ipV4 returns the string representation of an IPv4 address in a /proc/net table.
func ipV4(ip net.IP, port int) string {
	return fmt.Sprintf("%08X:%04X", reverse(ip.To4()), port)
}

// ipV6 returns the string representation of an IPv6 address in a /proc/net table.
func ipV6(ip net.IP, port int) string {
	return fmt.Sprintf("%032X:%04X", reverse(ip.To16()), port)
}

func reverse(b []byte) []byte {
	c := make([]byte, len(b))
	for i, e := range b {
		c[len(b)-1-i] = e
	}
	return c
}

func contains(b []byte, addr []string, addrIsUnspecified []bool) bool {
	for i, a := range addr {
		if addrIsUnspecified[i] {
			_, ap, pok := strings.Cut(a, ":")
			_, bp, bok := bytes.Cut(b, []byte(":"))
			if pok && bok && strings.EqualFold(string(bp), ap) {
				return true
			}
		} else if strings.EqualFold(string(b), a) {
			return true
		}
	}
	return false
}

func containsUnspecifiedAddr(addr []string) (yes bool, which []bool, bad []string) {
	which = make([]bool, len(addr))
	for i, a := range addr {
		prefix, _, ok := strings.Cut(a, ":")
		if !ok {
			continue
		}
		ip, err := hex.DecodeString(prefix)
		if err != nil {
			bad = append(bad, a)
		}
		if net.IP(ip).IsUnspecified() {
			yes = true
			which[i] = true
		}
	}
	return yes, which, bad
}

type netMetrics struct {
	done chan struct{}

	monitorRegistry *monitoring.Registry

	device          *monitoring.String // name of the device being monitored
	packets         *monitoring.Uint   // number of packets (events) processed (read)
	bytes           *monitoring.Uint   // number of bytes processed
	rxQueue         *monitoring.Uint   // value of the rx_queue field from /proc/net/udp{,6} (only on linux systems)
	arrivalPeriod   metrics.Sample     // histogram of the elapsed time between packet arrivals
	processingTime  metrics.Sample     // histogram of the elapsed time between packet receipt and publication
	eventsPublished *monitoring.Uint   // number of events published
	lastPacket      time.Time
}

func newNetMetrics(reg *monitoring.Registry) netMetrics {
	return netMetrics{
		monitorRegistry: reg,

		device:          monitoring.NewString(reg, "device"),
		packets:         monitoring.NewUint(reg, "received_events_total"),
		bytes:           monitoring.NewUint(reg, "received_bytes_total"),
		rxQueue:         monitoring.NewUint(reg, "receive_queue_length"),
		eventsPublished: monitoring.NewUint(reg, "published_events_total"),
		arrivalPeriod:   metrics.NewUniformSample(1024),
		processingTime:  metrics.NewUniformSample(1024),
	}
}

// EventReceived update all metrics related to receiving events.
// The metrics are:
//   - Events (packets) count
//   - Arrival period
//   - Bytes read/processed
func (n *netMetrics) EventReceived(len int, timestamp time.Time) {
	if n == nil {
		return
	}

	if !n.lastPacket.IsZero() {
		n.arrivalPeriod.Update(timestamp.Sub(n.lastPacket).Nanoseconds())
	}

	n.lastPacket = timestamp

	n.packets.Add(1)
	n.bytes.Add(uint64(len)) // nolint:gosec // length is never negative
}

// EventPublished updates all metrics related to published events.
// The metrics are:
//   - Published events count
//   - Event processing (publishing) time
func (n *netMetrics) EventPublished(start time.Time) {
	if n == nil {
		return
	}
	n.processingTime.Update(time.Since(start).Nanoseconds())
	n.eventsPublished.Inc()
}

// Registry returns the monitoring registry of the metricset.
func (m *netMetrics) Registry() *monitoring.Registry {
	if m == nil {
		return nil
	}

	return m.monitorRegistry
}

// Close closes the metricset and unregister the metrics.
func (m *netMetrics) Close() {
	if m == nil {
		return
	}
	if m.done != nil {
		// Shut down poller and wait until done before unregistering metrics.
		m.done <- struct{}{}
	}

	m.monitorRegistry = nil
}
