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
	"sync"
	"time"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/harvester"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/filebeat/inputsource/common/streaming"
	"github.com/elastic/beats/v7/filebeat/inputsource/tcp"
	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func init() {
	err := input.Register("tcp", NewInput)
	if err != nil {
		panic(err)
	}
}

// Input for TCP connection
type Input struct {
	mutex   sync.Mutex
	server  *tcp.Server
	started bool
	outlet  channel.Outleter
	config  *config
	log     *logp.Logger
}

// NewInput creates a new TCP input
func NewInput(
	cfg *conf.C,
	connector channel.Connector,
	context input.Context,
) (input.Input, error) {
	out, err := connector.Connect(cfg)
	if err != nil {
		return nil, err
	}

	forwarder := harvester.NewForwarder(out)

	config := defaultConfig
	err = cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}

	cb := func(data []byte, metadata inputsource.NetworkMetadata) {
		event := createEvent(data, metadata)
		forwarder.Send(event)
	}

	splitFunc, err := streaming.SplitFunc(config.Framing, []byte(config.LineDelimiter))
	if err != nil {
		return nil, err
	}

	logger := logp.NewLogger("input.tcp").With("address", config.Config.Host)
	factory := streaming.SplitHandlerFactory(inputsource.FamilyTCP, logger, tcp.MetadataCallback, cb, splitFunc)

	server, err := tcp.New(&config.Config, factory)
	if err != nil {
		return nil, err
	}

	return &Input{
		server:  server,
		started: false,
		outlet:  out,
		config:  &config,
		log:     logger,
	}, nil
}

// Run start a TCP input
func (p *Input) Run() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if !p.started {
		p.log.Info("Starting TCP input")
		err := p.server.Start()
		if err != nil {
			p.log.Errorw("Error starting the TCP server", "error", err)
		}
		p.started = true
	}
}

// Stop stops TCP server
func (p *Input) Stop() {
	defer p.outlet.Close()
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.log.Info("Stopping TCP input")
	p.server.Stop()
	p.started = false
}

// Wait stop the current server
func (p *Input) Wait() {
	p.Stop()
}

func createEvent(raw []byte, metadata inputsource.NetworkMetadata) beat.Event {
	return beat.Event{
		Timestamp: time.Now(),
		Fields: mapstr.M{
			"message": string(raw),
			"log": mapstr.M{
				"source": mapstr.M{
					"address": metadata.RemoteAddr.String(),
				},
			},
		},
	}
}
