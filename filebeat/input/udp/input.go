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

	"github.com/elastic/beats/v7/filebeat/input/netmetrics"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/filebeat/inputsource/udp"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
	metrics := netmetrics.NewUDP("udp", ctx.ID, s.config.Host, uint64(s.config.ReadBuffer), pollInterval, log)
	defer metrics.Close()

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
		metrics.Log(data, evt.Timestamp)
	})

	log.Debug("udp input initialized")

	err := server.Run(ctxtool.FromCanceller(ctx.Cancelation))
	// Ignore error from 'Run' in case shutdown was signaled.
	if ctxerr := ctx.Cancelation.Err(); ctxerr != nil {
		err = ctxerr
	}
	return err
}
