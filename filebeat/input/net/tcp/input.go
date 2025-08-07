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
	"fmt"
	"net"
	"time"

	"github.com/dustin/go-humanize"

	netinput "github.com/elastic/beats/v7/filebeat/input/net"
	"github.com/elastic/beats/v7/filebeat/input/netmetrics"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/filebeat/inputsource/common/streaming"
	"github.com/elastic/beats/v7/filebeat/inputsource/tcp"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/go-concert/ctxtool"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

func Plugin() input.Plugin {
	return input.Plugin{
		Name:       "tcp",
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "tcp packet server",
		Manager:    netinput.NewManager(configure),
	}
}

func configure(cfg *conf.C) (netinput.Input, error) {
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
	metrics *netmetrics.TCP
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
	l, err := net.Listen("tcp", s.Host)
	if err != nil {
		return err
	}
	return l.Close()
}

// InitMetrics initalises and returns an netmetrics.TCP
func (s *server) InitMetrics(id string, reg *monitoring.Registry, logger *logp.Logger) netinput.Metrics {
	s.metrics = netmetrics.NewTCP(reg, s.Host, time.Minute, logger)
	return s.metrics
}

// Run runs the input
func (s *server) Run(ctx input.Context, evtChan chan<- netinput.DataMetadata, m netinput.Metrics) (err error) {
	defer s.metrics.Close()

	split, err := streaming.SplitFunc(s.Framing, []byte(s.LineDelimiter))
	if err != nil {
		ctx.UpdateStatus(status.Failed, "Failed to configure split function: "+err.Error())
		return err
	}

	server, err := tcp.New(
		&s.Config,
		streaming.SplitHandlerFactory(
			inputsource.FamilyTCP,
			ctx.Logger,
			tcp.MetadataCallback,
			func(data []byte, metadata inputsource.NetworkMetadata) {
				now := time.Now()
				m.EventReceived(len(data), now)
				ctx.Logger.Debugw(
					"Data received",
					"bytes", len(data),
					"remote_address", metadata.RemoteAddr.String(),
					"truncated", metadata.Truncated)

				evtChan <- netinput.DataMetadata{
					Data:      data,
					Metadata:  metadata,
					Timestamp: now,
				}
			},
			split,
		),
		ctx.Logger,
	)
	if err != nil {
		return fmt.Errorf("Failed to start TCP server: %w", err)
	}

	ctx.Logger.Debug("tcp input initialized")
	ctx.UpdateStatus(status.Running, "")

	return server.Run(ctxtool.FromCanceller(ctx.Cancelation))
}
