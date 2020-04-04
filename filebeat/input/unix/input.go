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
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/v7/filebeat/channel"
	"github.com/elastic/beats/v7/filebeat/harvester"
	"github.com/elastic/beats/v7/filebeat/input"
	"github.com/elastic/beats/v7/filebeat/inputsource"
	netcommon "github.com/elastic/beats/v7/filebeat/inputsource/common"
	"github.com/elastic/beats/v7/filebeat/inputsource/unix"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

func init() {
	err := input.Register("unix", NewInput)
	if err != nil {
		panic(err)
	}
}

// Input for Unix socket connection
type Input struct {
	sync.Mutex
	server  *unix.Server
	started bool
	outlet  channel.Outleter
	config  *config
	log     *logp.Logger
}

// NewInput creates a new Unix socket input
func NewInput(
	cfg *common.Config,
	connector channel.Connector,
	context input.Context,
) (input.Input, error) {

	out, err := connector.ConnectWith(cfg, beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			DynamicFields: context.DynamicFields,
		},
	})
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
		forwarder.Send(beat.Event{
			Timestamp: time.Now(),
			Fields: common.MapStr{
				"message": string(data),
			},
		})
	}

	splitFunc := netcommon.SplitFunc([]byte(config.LineDelimiter))
	if splitFunc == nil {
		return nil, fmt.Errorf("unable to create splitFunc for delimiter %s", config.LineDelimiter)
	}

	factory := netcommon.SplitHandlerFactory(netcommon.FamilyUnix, unix.MetadataCallback, cb, splitFunc)

	server, err := unix.New(&config.Config, factory)
	if err != nil {
		return nil, err
	}

	return &Input{
		server:  server,
		started: false,
		outlet:  out,
		config:  &config,
		log:     logp.NewLogger("unix input").With("path", config.Config.Path),
	}, nil
}

// Run start a Unix socket input
func (p *Input) Run() {
	p.Lock()
	defer p.Unlock()

	if !p.started {
		p.log.Info("Starting Unix socket input")
		err := p.server.Start()
		if err != nil {
			p.log.Errorw("Error starting the Unix socket server", "error", err)
		}
		p.started = true
	}
}

// Stop stops Unix socket server
func (p *Input) Stop() {
	defer p.outlet.Close()
	p.Lock()
	defer p.Unlock()

	p.log.Info("Stopping Unix socket input")
	p.server.Stop()
	p.started = false
}

// Wait stop the current server
func (p *Input) Wait() {
	p.Stop()
}
