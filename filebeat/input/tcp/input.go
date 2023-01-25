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
	"encoding/binary"
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
				evt.Fields["log"] = mapstr.M{"source": logSource(metadata.RemoteAddr)}
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

// logSource returns the source fields for the provided log source metadata address.
func logSource(src net.Addr) mapstr.M {
	addr := src.String()
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return mapstr.M{"address": addr}
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		return mapstr.M{"address": addr}
	}
	m := mapstr.M{
		"address": host,
		"port":    p,
	}
	// Check whether host parses as an IP address, but use the
	// original string to avoid unnecessary allocs for String.
	if net.ParseIP(host) != nil {
		m["ip"] = host
	}
	return m
}

// inputMetrics handles the input's metric reporting.
type inputMetrics struct {
	unregister func()
	done       chan struct{}

	lastPacket time.Time

	device         *monitoring.String // name of the device being monitored
	packets        *monitoring.Uint   // number of packets processed
	bytes          *monitoring.Uint   // number of bytes processed
	rxQueue        *monitoring.Uint   // value of the rx_queue field from /proc/net/tcp (only on linux systems)
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
		host, port, ok := strings.Cut(device, ":")
		if !ok {
			log.Warnf("failed to get address for %s: no port separator", device)
			return out
		}
		ip, err := net.LookupIP(host)
		if err != nil {
			log.Warnf("failed to get address for %s: %v", device, err)
			return out
		}
		p, err := strconv.ParseInt(port, 10, 16)
		if err != nil {
			log.Warnf("failed to get port for %s: %v", device, err)
			return out
		}
		ph := strconv.FormatInt(p, 16)
		addr := make([]string, 0, len(ip))
		for _, p := range ip {
			p4 := p.To4()
			if len(p4) != net.IPv4len {
				continue
			}
			addr = append(addr, fmt.Sprintf("%X:%s", binary.LittleEndian.Uint32(p4), ph))
		}
		out.done = make(chan struct{})
		go out.poll(addr, poll, log)
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

// poll periodically gets TCP buffer stats from the OS.
func (m *inputMetrics) poll(addr []string, each time.Duration, log *logp.Logger) {
	t := time.NewTicker(each)
	for {
		select {
		case <-t.C:
			rx, err := procNetTCP(addr)
			if err != nil {
				log.Warnf("failed to get tcp stats from /proc: %v", err)
				continue
			}
			m.rxQueue.Set(uint64(rx))
		case <-m.done:
			t.Stop()
			return
		}
	}
}

// procNetTCP returns the rx_queue field of the TCP socket table for the
// socket on the provided address formatted in hex, xxxxxxxx:xxxx.
// This function is only useful on linux due to its dependence on the /proc
// filesystem, but is kept in this file for simplicity.
func procNetTCP(addr []string) (rx int64, err error) {
	b, err := os.ReadFile("/proc/net/tcp")
	if err != nil {
		return 0, err
	}
	lines := bytes.Split(b, []byte("\n"))
	if len(lines) < 2 {
		return 0, fmt.Errorf("/proc/net/tcp entry not found for %s (no line)", addr)
	}
	for _, l := range lines[1:] {
		f := bytes.Fields(l)
		if contains(f[1], addr) {
			_, r, ok := bytes.Cut(f[4], []byte(":"))
			if !ok {
				return 0, errors.New("no rx_queue field " + string(f[4]))
			}
			rx, err = strconv.ParseInt(string(r), 16, 64)
			if err != nil {
				return 0, fmt.Errorf("failed to parse rx_queue: %w", err)
			}
			return rx, nil
		}
	}
	return 0, fmt.Errorf("/proc/net/tcp entry not found for %s", addr)
}

func contains(b []byte, addr []string) bool {
	for _, a := range addr {
		if strings.EqualFold(string(b), a) {
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
