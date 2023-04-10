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

package unix

import (
	"net"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/rcrowley/go-metrics"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/filebeat/inputsource/unix"
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
		Name:       "unix",
		Stability:  feature.Beta,
		Deprecated: false,
		Info:       "unix socket server",
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

type config struct {
	unix.Config `config:",inline"`
}

func defaultConfig() config {
	return config{
		Config: unix.Config{
			Timeout:        time.Minute * 5,
			MaxMessageSize: 20 * humanize.MiByte,
			SocketType:     unix.StreamSocket,
			LineDelimiter:  "\n",
		},
	}
}

type server struct {
	unix.Server
	config
}

func newServer(config config) (*server, error) {
	return &server{config: config}, nil
}

func (s *server) Name() string { return "unix" }

func (s *server) Test(_ input.TestContext) error {
	l, err := net.Listen("unix", s.config.Config.Path)
	if err != nil {
		return err
	}
	return l.Close()
}

func (s *server) Run(ctx input.Context, publisher stateless.Publisher) error {
	log := ctx.Logger.With("path", s.config.Config.Path)

	log.Info("Starting Unix socket input")
	defer log.Info("Unix socket input stopped")

	metrics := newInputMetrics(ctx.ID, s.config.Path, log)
	defer metrics.close()

	server, err := unix.New(log, &s.config.Config, func(data []byte, _ inputsource.NetworkMetadata) {
		evt := beat.Event{
			Timestamp: time.Now(),
			Fields: mapstr.M{
				"message": string(data),
			},
		}
		publisher.Publish(evt)

		// This must be called after publisher.Publish to measure
		// the processing time metric.
		metrics.log(data, evt.Timestamp)
	})
	if err != nil {
		return err
	}

	log.Debugf("%s Input '%v' initialized", s.config.Config.SocketType, ctx.ID)

	err = server.Run(ctxtool.FromCanceller(ctx.Cancelation))

	// ignore error from 'Run' in case shutdown was signaled.
	if ctxerr := ctx.Cancelation.Err(); ctxerr != nil {
		err = ctxerr
	}
	return err
}

// inputMetrics handles the input's metric reporting.
type inputMetrics struct {
	unregister func()

	lastPacket time.Time

	path           *monitoring.String // name of the socket path being monitored
	packets        *monitoring.Uint   // number of packets processed
	bytes          *monitoring.Uint   // number of bytes processed
	arrivalPeriod  metrics.Sample     // histogram of the elapsed time between packet arrivals
	processingTime metrics.Sample     // histogram of the elapsed time between packet receipt and publication
}

// newInputMetrics returns an input metric for the unix socket processor. If id is empty
// a nil inputMetric is returned.
func newInputMetrics(id, path string, log *logp.Logger) *inputMetrics {
	if id == "" {
		return nil
	}
	reg, unreg := inputmon.NewInputRegistry("unix", id, nil)
	out := &inputMetrics{
		unregister:     unreg,
		path:           monitoring.NewString(reg, "path"),
		packets:        monitoring.NewUint(reg, "received_events_total"),
		bytes:          monitoring.NewUint(reg, "received_bytes_total"),
		arrivalPeriod:  metrics.NewUniformSample(1024),
		processingTime: metrics.NewUniformSample(1024),
	}
	_ = adapter.NewGoMetrics(reg, "arrival_period", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.arrivalPeriod))
	_ = adapter.NewGoMetrics(reg, "processing_time", adapter.Accept).
		Register("histogram", metrics.NewHistogram(out.processingTime))

	out.path.Set(path)

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

func (m *inputMetrics) close() {
	if m == nil {
		return
	}
	m.unregister()
}
