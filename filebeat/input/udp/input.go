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
	"fmt"
	"net"
	"runtime/debug"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/elastic/beats/v7/filebeat/input/netmetrics"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	stateless "github.com/elastic/beats/v7/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/filebeat/inputsource/udp"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/management/status"

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
		Manager:    v2.ConfigureWith(configure),
	}
}

func configure(cfg *conf.C) (v2.Input, error) {
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
	evtChan chan beat.Event
}

type config struct {
	udp.Config `config:",inline"`
}

func newServer(config config) (*server, error) {
	return &server{
		config:  config,
		evtChan: make(chan beat.Event),
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

func (s *server) publishLoop(ctx input.Context, publisher stateless.Publisher, metrics *netmetrics.UDP) {
	logger := ctx.Logger
	logger.Debug("starting publish loop")
	defer logger.Debug("finished publish loop")
	for {
		select {
		case <-ctx.Cancelation.Done():
			logger.Debug("Context cancelled, closing publish Loop")
			return
		case evt := <-s.evtChan:
			start := time.Now()
			publisher.Publish(evt)
			metrics.EventPublished(start)
		}
	}
}

func (s *server) Run(ctx input.Context, pipeline beat.PipelineConnector) (err error) {
	log := ctx.Logger.With("host", s.Host)

	defer func() {
		if v := recover(); v != nil {
			if e, ok := v.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("UDP input panic with: %+v\n%s", v, debug.Stack())
			}
			log.Errorw("UDP input panic", err)
		}
	}()

	publisher, _ := pipeline.Connect()

	log.Info("starting udp socket input")
	defer log.Info("udp input stopped")

	ctx.UpdateStatus(status.Starting, "")
	ctx.UpdateStatus(status.Configuring, "")

	const pollInterval = time.Minute
	// #nosec G115 -- ignore "overflow conversion int64 -> uint64", config validation ensures value is always positive.
	metrics := netmetrics.NewUDP("udp", ctx.ID, s.Host, uint64(s.ReadBuffer), pollInterval, log)
	defer metrics.Close()

	go s.publishLoop(ctx, publisher, metrics)

	server := udp.New(&s.Config, func(data []byte, metadata inputsource.NetworkMetadata) {
		now := time.Now()
		metrics.EventReceived(len(data), now)
		log.Debugw(
			"Data received",
			"bytes", len(data),
			"remote_address", metadata.RemoteAddr.String(),
			"truncated", metadata.Truncated,
		)
		evt := beat.Event{
			Timestamp: now,
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
		s.evtChan <- evt
	}, log)

	log.Debug("udp input initialized")
	ctx.UpdateStatus(status.Running, "")

	err = server.Run(ctxtool.FromCanceller(ctx.Cancelation))
	// Ignore error from 'Run' in case shutdown was signaled.
	if ctxerr := ctx.Cancelation.Err(); ctxerr != nil {
		err = ctxerr
	}

	if err != nil {
		ctx.UpdateStatus(status.Failed, "Input exited unexpectedly: "+err.Error())
	} else {
		ctx.UpdateStatus(status.Stopped, "")
	}

	return err
}
