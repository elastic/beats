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

package stdin

import (
	"fmt"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/filebeat/input/log"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	err := input.Register("stdin", NewInput)
	if err != nil {
		panic(err)
	}
}

// Input is an input for stdin
type Input struct {
	harvester *log.Harvester
	started   bool
	cfg       *common.Config
	outlet    channel.Outleter
	registry  *harvester.Registry
}

// NewInput creates a new stdin input
// This input contains one harvester which is reading from stdin
func NewInput(cfg *common.Config, outlet channel.Connector, context input.Context) (input.Input, error) {
	out, err := outlet.ConnectWith(cfg, beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			DynamicFields: context.DynamicFields,
		},
	})
	if err != nil {
		return nil, err
	}

	p := &Input{
		started:  false,
		cfg:      cfg,
		outlet:   out,
		registry: harvester.NewRegistry(),
	}

	p.harvester, err = p.createHarvester(file.State{Source: "-"})
	if err != nil {
		return nil, fmt.Errorf("Error initializing stdin harvester: %v", err)
	}

	return p, nil
}

// Run runs the input
func (p *Input) Run() {
	// Make sure stdin harvester is only started once
	if !p.started {
		err := p.harvester.Setup()
		if err != nil {
			logp.Err("Error setting up stdin harvester: %s", err)
			return
		}
		if err = p.registry.Start(p.harvester); err != nil {
			logp.Err("Error starting the harvester: %s", err)
		}
		p.started = true
	}
}

// createHarvester creates a new harvester instance from the given state
func (p *Input) createHarvester(state file.State) (*log.Harvester, error) {
	// Each harvester gets its own copy of the outlet
	h, err := log.NewHarvester(
		p.cfg,
		state, nil, nil,
		func() channel.Outleter {
			return p.outlet
		},
	)

	return h, err
}

// Wait waits for completion of the input.
func (p *Input) Wait() {}

// Stop stops the input
func (p *Input) Stop() {
	p.outlet.Close()
}
