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

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/filebeat/inputsource/common/streaming"
	"github.com/elastic/beats/v7/filebeat/inputsource/tcp"
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
		Name:       "tcp",
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "tcp packet server",
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
		Config: tcp.Config{
			Timeout:        time.Minute * 5,
			MaxMessageSize: 20 * humanize.MiByte,
		},
		LineDelimiter: "\n",
	}
}

type server struct {
	tcp.Server
	config
}

type config struct {
	tcp.Config `config:",inline"`

	LineDelimiter string                `config:"line_delimiter" validate:"nonzero"`
	Framing       streaming.FramingType `config:"framing"`
}

func newServer(config config) (*server, error) {
	return &server{config: config}, nil
}

func (s *server) Name() string { return "tcp" }

func (s *server) Test(_ input.TestContext) error {
	l, err := net.Listen("tcp", s.config.Config.Host)
	if err != nil {
		return err
	}
	return l.Close()
}

func (s *server) Run(ctx input.Context, publisher stateless.Publisher) error {
	log := ctx.Logger.With("host", s.config.Config.Host)

	log.Info("starting tcp socket input")
	defer log.Info("tcp input stopped")

	const pollInterval = time.Minute
	metrics := newInputMetrics(ctx.ID, s.config.Host, pollInterval, log)
	defer metrics.close()

	split, err := streaming.SplitFunc(s.config.Framing, []byte(s.config.LineDelimiter))
	if err != nil {
		return err
	}

	server, err := tcp.New(&s.config.Config, streaming.SplitHandlerFactory(
		inputsource.FamilyTCP, log, tcp.MetadataCallback, func(data []byte, metadata inputsource.NetworkMetadata) {
			evt := beat.Event{
				Timestamp: time.Now(),
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
		},
		split,
	))
	if err != nil {
		return err
	}

	log.Debug("tcp input initialized")

	err = server.Run(ctxtool.FromCanceller(ctx.Cancelation))
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
	rxQueue        *monitoring.Uint   // value of the rx_queue field from /proc/net/tcp{,6} (only on linux systems)
	arrivalPeriod  metrics.Sample     // histogram of the elapsed time between packet arrivals
	processingTime metrics.Sample     // histogram of the elapsed time between packet receipt and publication
}

// newInputMetrics returns an input metric for the TCP processor. If id is empty
// a nil inputMetric is returned.
func newInputMetrics(id, device string, poll time.Duration, log *logp.Logger) *inputMetrics {
	if id == "" {
		return nil
	}
	reg, unreg := inputmon.NewInputRegistry("tcp", id, nil)
	out := &inputMetrics{
		unregister:     unreg,
		device:         monitoring.NewString(reg, "device"),
		packets:        monitoring.NewUint(reg, "received_events_total"),
		bytes:          monitoring.NewUint(reg, "received_bytes_total"),
		rxQueue:        monitoring.NewUint(reg, "receive_queue_length"),
		arrivalPeriod:  metrics.NewUniformSample(1024),
		processingTime: metrics.NewUniformSample(1024),
	}
	_ = adapter.NewGoMetrics(reg, "arrival_period", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.arrivalPeriod))
	_ = adapter.NewGoMetrics(reg, "processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.processingTime))

	out.device.Set(device)

	if poll > 0 && runtime.GOOS == "linux" {
		host, port, err := net.SplitHostPort(device)
		if err != nil {
			log.Warnf("failed to get address for %s: could not split host and port:", err)
			return out
		}
		ip, err := net.LookupIP(host)
		if err != nil {
			log.Warnf("failed to get address for %s: %v", device, err)
			return out
		}
		pn, err := strconv.ParseInt(port, 10, 16)
		if err != nil {
			log.Warnf("failed to get port for %s: %v", device, err)
			return out
		}
		addr := make([]string, 0, len(ip))
		addr6 := make([]string, 0, len(ip))
		for _, p := range ip {
			switch len(p) {
			case net.IPv4len:
				addr = append(addr, ipv4KernelAddr(p, int(pn)))
			case net.IPv6len:
				addr6 = append(addr6, ipv6KernelAddr(p, int(pn)))
			default:
				log.Warnf("unexpected addr length %d for %s", len(p), p)
			}
		}
		out.done = make(chan struct{})
		go out.poll(addr, addr6, poll, log)
	}

	return out
}

func ipv4KernelAddr(ip net.IP, port int) string {
	return fmt.Sprintf("%08X:%04X", reverse(ip.To4()), port)
}

func ipv6KernelAddr(ip net.IP, port int) string {
	return fmt.Sprintf("%032X:%04X", reverse(ip.To16()), port)
}

func reverse(b []byte) []byte {
	c := make([]byte, len(b))
	for i, e := range b {
		c[len(b)-1-i] = e
	}
	return c
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

// poll periodically gets TCP buffer stats from the OS.
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
	// base level for the rx_queue values and ensures that if the
	// constructed address values are malformed we panic early
	// within the period of system testing.
	rx, err := procNetTCP("/proc/net/tcp", addr, hasUnspecified, addrIsUnspecified)
	if err != nil {
		log.Warnf("failed to get initial tcp stats from /proc: %v", err)
	}
	rx6, err := procNetTCP("/proc/net/tcp6", addr6, hasUnspecified6, addrIsUnspecified6)
	if err != nil {
		log.Warnf("failed to get initial tcp6 stats from /proc: %v", err)
	}
	m.rxQueue.Set(uint64(rx + rx6))

	t := time.NewTicker(each)
	for {
		select {
		case <-t.C:
			rx, err := procNetTCP("/proc/net/tcp", addr, hasUnspecified, addrIsUnspecified)
			if err != nil {
				log.Warnf("failed to get tcp stats from /proc: %v", err)
				continue
			}
			rx6, err := procNetTCP("/proc/net/tcp6", addr6, hasUnspecified6, addrIsUnspecified6)
			if err != nil {
				log.Warnf("failed to get tcp6 stats from /proc: %v", err)
				continue
			}
			m.rxQueue.Set(uint64(rx + rx6))
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

// procNetTCP returns the rx_queue field of the TCP socket table for the
// socket on the provided address formatted in hex, xxxxxxxx:xxxx or the IPv6
// equivalent.
// This function is only useful on linux due to its dependence on the /proc
// filesystem, but is kept in this file for simplicity. If hasUnspecified
// is true, all addresses listed in the file in path are considered, and the
// sum of rx_queue matching the addr ports is returned where the corresponding
// addrIsUnspecified is true.
func procNetTCP(path string, addr []string, hasUnspecified bool, addrIsUnspecified []bool) (rx int64, err error) {
	if len(addr) == 0 {
		return 0, nil
	}
	if len(addr) != len(addrIsUnspecified) {
		return 0, errors.New("mismatched address/unspecified lists: please report this")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	lines := bytes.Split(b, []byte("\n"))
	if len(lines) < 2 {
		return 0, fmt.Errorf("%s entry not found for %s (no line)", path, addr)
	}
	var found bool
	for _, l := range lines[1:] {
		f := bytes.Fields(l)
		const queuesField = 4
		if len(f) > queuesField && contains(f[1], addr, addrIsUnspecified) {
			_, r, ok := bytes.Cut(f[4], []byte(":"))
			if !ok {
				return 0, errors.New("no rx_queue field " + string(f[queuesField]))
			}
			found = true

			// queue lengths are decimal, e.g.:
			// - https://elixir.bootlin.com/linux/v6.2.11/source/net/ipv4/tcp_ipv4.c#L2643
			// - https://elixir.bootlin.com/linux/v6.2.11/source/net/ipv6/tcp_ipv6.c#L1987
			v, err := strconv.ParseInt(string(r), 10, 64)
			if err != nil {
				return 0, fmt.Errorf("failed to parse rx_queue: %w", err)
			}
			rx += v

			if hasUnspecified {
				continue
			}
			return rx, nil
		}
	}
	if found {
		return rx, nil
	}
	return 0, fmt.Errorf("%s entry not found for %s", path, addr)
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
