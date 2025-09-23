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
	"net"
	"time"

	"github.com/dustin/go-humanize"

	netinput "github.com/elastic/beats/v7/filebeat/input/net"
	"github.com/elastic/beats/v7/filebeat/input/netmetrics"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/filebeat/inputsource/udp"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/management/status"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/go-concert/ctxtool"
)

func Plugin() input.Plugin {
	return input.Plugin{
		Name:       "udp",
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "udp packet server",
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
	metrics *netmetrics.UDP
}

type config struct {
	udp.Config `config:",inline"`
}

func newServer(config config) (*server, error) {
	return &server{
		config: config,
	}, nil
}

func (s *server) Name() string { return "udp" }

func (s *server) Test(_ input.TestContext) error {
	l, err := net.Listen("udp", s.Host)
	if err != nil {
		return err
	}
	return l.Close()
}

func (s *server) InitMetrics(id string, reg *monitoring.Registry, logger *logp.Logger) netinput.Metrics {
	s.metrics = netmetrics.NewUDP(reg, s.Host, uint64(s.ReadBuffer), time.Second, logger)
	return s.metrics
}

func (s *server) Run(ctx input.Context, evtChan chan<- netinput.DataMetadata, metrics netinput.Metrics) (err error) {
	logger := ctx.Logger
	defer s.metrics.Close()

	server := udp.New(&s.Config, func(data []byte, metadata inputsource.NetworkMetadata) {
		now := time.Now()
		metrics.EventReceived(len(data), now)
		logger.Debugw(
			"Data received",
			"bytes", len(data),
			"remote_address", metadata.RemoteAddr.String(),
			"truncated", metadata.Truncated)

		evtChan <- netinput.DataMetadata{
			Data:      data,
			Metadata:  metadata,
			Timestamp: now,
		}
	}, logger)

	logger.Debug("udp input initialized")
	ctx.UpdateStatus(status.Running, "")

	return server.Run(ctxtool.FromCanceller(ctx.Cancelation))
}
