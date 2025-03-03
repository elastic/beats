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

package netmetrics

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
)

// UDP captures UDP related metrics.
type UDP struct {
	unregister func()
	done       chan struct{}

	monitorRegistry *monitoring.Registry

	lastPacket time.Time

	device         *monitoring.String // name of the device being monitored
	packets        *monitoring.Uint   // number of packets processed
	bytes          *monitoring.Uint   // number of bytes processed
	bufferLen      *monitoring.Uint   // configured read buffer length
	rxQueue        *monitoring.Uint   // value of the rx_queue field from /proc/net/udp{,6} (only on linux systems)
	drops          *monitoring.Uint   // number of udp drops noted in /proc/net/udp{,6}
	arrivalPeriod  metrics.Sample     // histogram of the elapsed time between packet arrivals
	processingTime metrics.Sample     // histogram of the elapsed time between packet receipt and publication
}

// NewUDP returns a new UDP input metricset. Note that if the id is empty then a nil UDP metricset is returned.
func NewUDP(inputName string, id string, device string, buflen uint64, poll time.Duration, log *logp.Logger) *UDP {
	if id == "" {
		return nil
	}
	reg, unreg := inputmon.NewInputRegistry(inputName, id, nil)
	out := &UDP{
		unregister:      unreg,
		monitorRegistry: reg,
		bufferLen:       monitoring.NewUint(reg, "udp_read_buffer_length_gauge"),
		device:          monitoring.NewString(reg, "device"),
		packets:         monitoring.NewUint(reg, "received_events_total"),
		bytes:           monitoring.NewUint(reg, "received_bytes_total"),
		rxQueue:         monitoring.NewUint(reg, "receive_queue_length"),
		drops:           monitoring.NewUint(reg, "system_packet_drops"),
		arrivalPeriod:   metrics.NewUniformSample(1024),
		processingTime:  metrics.NewUniformSample(1024),
	}
	_ = adapter.NewGoMetrics(reg, "arrival_period", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.arrivalPeriod))
	_ = adapter.NewGoMetrics(reg, "processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.processingTime))

	out.device.Set(device)
	out.bufferLen.Set(buflen)

	if poll > 0 && runtime.GOOS == "linux" {
		addr, addr6, err := addrs(device, log)
		if err != nil {
			log.Warn(err)
			return out
		}
		out.done = make(chan struct{})
		go out.poll(addr, addr6, poll, log)
	}

	return out
}

// Log logs metric for the given packet.
func (m *UDP) Log(data []byte, timestamp time.Time) {
	if m == nil {
		return
	}
	m.processingTime.Update(time.Since(timestamp).Nanoseconds())
	m.packets.Add(1)
	m.bytes.Add(uint64(len(data)))
	if !m.lastPacket.IsZero() {
		m.arrivalPeriod.Update(timestamp.Sub(m.lastPacket).Nanoseconds())
	}
	m.lastPacket = timestamp
}

// poll periodically gets UDP buffer and packet drops stats from the OS.
func (m *UDP) poll(addr, addr6 []string, each time.Duration, log *logp.Logger) {
	hasUnspecified, addrIsUnspecified, badAddr := containsUnspecifiedAddr(addr)
	if badAddr != nil {
		log.Warnf("failed to parse IPv4 addrs for metric collection %q", badAddr)
	}
	hasUnspecified6, addrIsUnspecified6, badAddr := containsUnspecifiedAddr(addr6)
	if badAddr != nil {
		log.Warnf("failed to parse IPv6 addrs for metric collection %q", badAddr)
	}

	// Do an initial check for access to the filesystem and of the
	// value constructed by containsUnspecifiedAddr. This gives a
	// base level for the rx_queue and drops values and ensures that
	// if the constructed address values are malformed we panic early
	// within the period of system testing.
	want4 := true
	rx, drops, err := procNetUDP("/proc/net/udp", addr, hasUnspecified, addrIsUnspecified)
	if err != nil {
		want4 = false
		log.Infof("did not get initial udp stats from /proc: %v", err)
	}
	want6 := true
	rx6, drops6, err := procNetUDP("/proc/net/udp6", addr6, hasUnspecified6, addrIsUnspecified6)
	if err != nil {
		want6 = false
		log.Infof("did not get initial udp6 stats from /proc: %v", err)
	}
	if !want4 && !want6 {
		log.Warnf("failed to get initial udp or udp6 stats from /proc: %v", err)
	} else {
		m.rxQueue.Set(uint64(rx + rx6))
		m.drops.Set(uint64(drops + drops6))
	}

	t := time.NewTicker(each)
	for {
		select {
		case <-t.C:
			var found bool
			rx, drops, err := procNetUDP("/proc/net/udp", addr, hasUnspecified, addrIsUnspecified)
			if err != nil {
				if want4 {
					log.Warnf("failed to get udp stats from /proc: %v", err)
				}
			} else {
				found = true
				want4 = true
			}
			rx6, drops6, err := procNetUDP("/proc/net/udp6", addr6, hasUnspecified6, addrIsUnspecified6)
			if err != nil {
				if want6 {
					log.Warnf("failed to get udp6 stats from /proc: %v", err)
				}
			} else {
				found = true
				want6 = true
			}
			if found {
				m.rxQueue.Set(uint64(rx + rx6))
				m.drops.Set(uint64(drops + drops6))
			}
		case <-m.done:
			t.Stop()
			return
		}
	}
}

// Registry returns the monitoring registry of the UDP metricset.
func (m *UDP) Registry() *monitoring.Registry {
	if m == nil {
		return nil
	}

	return m.monitorRegistry
}

// procNetUDP returns the rx_queue and drops field of the UDP socket table
// for the socket on the provided address formatted in hex, xxxxxxxx:xxxx or
// the IPv6 equivalent.
// This function is only useful on linux due to its dependence on the /proc
// filesystem, but is kept in this file for simplicity. If hasUnspecified
// is true, all addresses listed in the file in path are considered, and the
// sum of rx_queue and drops matching the addr ports is returned where the
// corresponding addrIsUnspecified is true.
func procNetUDP(path string, addr []string, hasUnspecified bool, addrIsUnspecified []bool) (rx, drops int64, err error) {
	if len(addr) == 0 {
		return 0, 0, nil
	}
	if len(addr) != len(addrIsUnspecified) {
		return 0, 0, errors.New("mismatched address/unspecified lists: please report this")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}
	lines := bytes.Split(b, []byte("\n"))
	if len(lines) < 2 {
		return 0, 0, fmt.Errorf("%s entry not found for %s (no line)", path, addr)
	}
	var found bool
	for _, l := range lines[1:] {
		f := bytes.Fields(l)
		const (
			queuesField = 4
			dropsField  = 12
		)
		if len(f) > dropsField && contains(f[1], addr, addrIsUnspecified) {
			_, r, ok := bytes.Cut(f[queuesField], []byte(":"))
			if !ok {
				return 0, 0, errors.New("no rx_queue field " + string(f[queuesField]))
			}
			found = true

			// queue lengths and drops are hex, e.g.:
			// - https://elixir.bootlin.com/linux/v6.2.11/source/net/ipv4/udp.c#L3110
			// - https://elixir.bootlin.com/linux/v6.2.11/source/net/ipv6/datagram.c#L1048
			v, err := strconv.ParseInt(string(r), 16, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("failed to parse rx_queue: %w", err)
			}
			rx += v

			v, err = strconv.ParseInt(string(f[dropsField]), 10, 64)
			if err != nil {
				return 0, 0, fmt.Errorf("failed to parse drops: %w", err)
			}
			drops += v

			if hasUnspecified {
				continue
			}
			return rx, drops, nil
		}
	}
	if found {
		return rx, drops, nil
	}
	return 0, 0, fmt.Errorf("%s entry not found for %s", path, addr)
}

// Close closes the UDP metricset and unregister the metrics.
func (m *UDP) Close() {
	if m == nil {
		return
	}
	if m.done != nil {
		// Shut down poller and wait until done before unregistering metrics.
		m.done <- struct{}{}
	}

	if m.unregister != nil {
		m.unregister()
		m.unregister = nil
	}

	m.monitorRegistry = nil
}
