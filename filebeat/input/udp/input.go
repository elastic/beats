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

package udp

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/rcrowley/go-metrics"

	"github.com/elastic/beats/v7/filebeat/input/internal/procnet"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/filebeat/inputsource/udp"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/monitoring/inputmon"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/monitoring/adapter"
	"github.com/elastic/go-concert/ctxtool"
)

func Plugin() input.Plugin {
	return input.Plugin{
		Name:       "udp",
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "udp packet server",
		Manager:    stateless.NewInputManager(configure),
	}
}

func configure(cfg *conf.C) (stateless.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	return newServer(config)
}

func defaultConfig() config {
	return config{
		Config: udp.Config{
			MaxMessageSize: 10 * humanize.KiByte,
			Host:           "localhost:8080",
			Timeout:        time.Minute * 5,
		},
	}
}

type server struct {
	udp.Server
	config
}

type config struct {
	udp.Config `config:",inline"`
}

func newServer(config config) (*server, error) {
	return &server{config: config}, nil
}

func (s *server) Name() string { return "udp" }

func (s *server) Test(_ input.TestContext) error {
	l, err := net.Listen("udp", s.config.Config.Host)
	if err != nil {
		return err
	}
	return l.Close()
}

func (s *server) Run(ctx input.Context, publisher stateless.Publisher) error {
	log := ctx.Logger.With("host", s.config.Config.Host)

	log.Info("starting udp socket input")
	defer log.Info("udp input stopped")

	const pollInterval = time.Minute
	metrics := newInputMetrics(ctx.ID, s.config.Host, uint64(s.config.ReadBuffer), pollInterval, log)
	defer metrics.close()

	server := udp.New(&s.config.Config, func(data []byte, metadata inputsource.NetworkMetadata) {
		evt := beat.Event{
			Timestamp: time.Now(),
			Meta: mapstr.M{
				"truncated": metadata.Truncated,
			},
			Fields: mapstr.M{
				"message": string(data),
			},
		}
		if metadata.RemoteAddr != nil {
			evt.Fields["log"] = mapstr.M{
				"source": mapstr.M{
					"address": metadata.RemoteAddr.String(),
				},
			}
		}

		publisher.Publish(evt)

		// This must be called after publisher.Publish to measure
		// the processing time metric.
		metrics.log(data, evt.Timestamp)
	})

	log.Debug("udp input initialized")

	err := server.Run(ctxtool.FromCanceller(ctx.Cancelation))
	// Ignore error from 'Run' in case shutdown was signaled.
	if ctxerr := ctx.Cancelation.Err(); ctxerr != nil {
		err = ctxerr
	}
	return err
}

// inputMetrics handles the input's metric reporting.
type inputMetrics struct {
	unregister func()
	done       chan struct{}

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

// newInputMetrics returns an input metric for the UDP processor. If id is empty
// a nil inputMetric is returned.
func newInputMetrics(id, device string, buflen uint64, poll time.Duration, log *logp.Logger) *inputMetrics {
	if id == "" {
		return nil
	}
	reg, unreg := inputmon.NewInputRegistry("udp", id, nil)
	out := &inputMetrics{
		unregister:     unreg,
		bufferLen:      monitoring.NewUint(reg, "udp_read_buffer_length_gauge"),
		device:         monitoring.NewString(reg, "device"),
		packets:        monitoring.NewUint(reg, "received_events_total"),
		bytes:          monitoring.NewUint(reg, "received_bytes_total"),
		rxQueue:        monitoring.NewUint(reg, "receive_queue_length"),
		drops:          monitoring.NewUint(reg, "system_packet_drops"),
		arrivalPeriod:  metrics.NewUniformSample(1024),
		processingTime: metrics.NewUniformSample(1024),
	}
	_ = adapter.NewGoMetrics(reg, "arrival_period", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.arrivalPeriod))
	_ = adapter.NewGoMetrics(reg, "processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.processingTime))

	out.device.Set(device)
	out.bufferLen.Set(buflen)

	if poll > 0 && runtime.GOOS == "linux" {
		addr, addr6, err := procnet.Addrs(device, log)
		if err != nil {
			log.Warn(err)
			return out
		}
		out.done = make(chan struct{})
		go out.poll(addr, addr6, poll, log)
	}

	return out
}

// log logs metric for the given packet.
func (m *inputMetrics) log(data []byte, timestamp time.Time) {
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
func (m *inputMetrics) poll(addr, addr6 []string, each time.Duration, log *logp.Logger) {
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
	rx, drops, err := procNetUDP("/proc/net/udp", addr, hasUnspecified, addrIsUnspecified)
	if err != nil {
		log.Warnf("failed to get initial udp stats from /proc: %v", err)
	}
	rx6, drops6, err := procNetUDP("/proc/net/udp6", addr6, hasUnspecified6, addrIsUnspecified6)
	if err != nil {
		log.Warnf("failed to get initial udp6 stats from /proc: %v", err)
	}
	m.rxQueue.Set(uint64(rx + rx6))
	m.drops.Set(uint64(drops + drops6))

	t := time.NewTicker(each)
	for {
		select {
		case <-t.C:
			rx, drops, err := procNetUDP("/proc/net/udp", addr, hasUnspecified, addrIsUnspecified)
			if err != nil {
				log.Warnf("failed to get udp stats from /proc: %v", err)
				continue
			}
			rx6, drops6, err := procNetUDP("/proc/net/udp6", addr6, hasUnspecified6, addrIsUnspecified6)
			if err != nil {
				log.Warnf("failed to get udp6 stats from /proc: %v", err)
				continue
			}
			m.rxQueue.Set(uint64(rx + rx6))
			m.drops.Set(uint64(drops + drops6))
		case <-m.done:
			t.Stop()
			return
		}
	}
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

			// queue lengths and drops are decimal, e.g.:
			// - https://elixir.bootlin.com/linux/v6.2.11/source/net/ipv4/udp.c#L3110
			// - https://elixir.bootlin.com/linux/v6.2.11/source/net/ipv6/datagram.c#L1048
			v, err := strconv.ParseInt(string(r), 10, 64)
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

func (m *inputMetrics) close() {
	if m == nil {
		return
	}
	if m.done != nil {
		// Shut down poller and wait until done before unregistering metrics.
		m.done <- struct{}{}
	}
	m.unregister()
}
