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

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/inputsource"
	"github.com/elastic/beats/filebeat/inputsource/tcp"
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	err := input.Register("tcp", NewInput)
	if err != nil {
		panic(err)
	}
}

// Input for TCP connection
type Input struct {
	sync.Mutex
	server  *tcp.Server
	started bool
	outlet  channel.Outleter
	config  *config
	log     *logp.Logger
}

// NewInput creates a new TCP input
func NewInput(
	cfg *common.Config,
	outlet channel.Connector,
	context input.Context,
) (input.Input, error) {
	cfgwarn.Experimental("TCP input type is used")

	out, err := outlet(cfg, context.DynamicFields)
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

	server, err := tcp.New(&config.Config, cb)
	if err != nil {
		return nil, err
	}

	return &Input{
		server:  server,
		started: false,
		outlet:  out,
		config:  &config,
		log:     logp.NewLogger("tcp input").With(config.Config.Host),
	}, nil
}

// Run start a TCP input
func (p *Input) Run() {
	p.Lock()
	defer p.Unlock()

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
	p.Lock()
	defer p.Unlock()

	p.log.Info("Stopping TCP input")
	p.server.Stop()
	p.started = false
}

// Wait stop the current server
func (p *Input) Wait() {
	p.Stop()
}

func createEvent(raw []byte, metadata inputsource.NetworkMetadata) *util.Data {
	data := util.NewData()
	data.Event = beat.Event{
		Timestamp: time.Now(),
		Fields: common.MapStr{
			"message": string(raw),
			"source":  metadata.RemoteAddr.String(),
		},
	}
	return data
}
