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

package beater

import (
	"fmt"
	"sync"
	"time"

	"github.com/elastic/beats/journalbeat/checkpoint"
	"github.com/elastic/beats/journalbeat/cmd/instance"
	"github.com/elastic/beats/journalbeat/input"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/journalbeat/config"
	_ "github.com/elastic/beats/journalbeat/include"

	// Add dedicated processors
	_ "github.com/elastic/beats/libbeat/processors/decode_csv_fields"
)

// Journalbeat instance
type Journalbeat struct {
	inputs []*input.Input
	done   chan struct{}
	config config.Config

	pipeline   beat.Pipeline
	checkpoint *checkpoint.Checkpoint
	logger     *logp.Logger
}

// New returns a new Journalbeat instance
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	cfgwarn.Experimental("Journalbeat is experimental.")

	config := config.DefaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	done := make(chan struct{})
	cp, err := checkpoint.NewCheckpoint(config.RegistryFile, 10, 1*time.Second)
	if err != nil {
		return nil, err
	}

	instance.SetupJournalMetrics()

	var inputs []*input.Input
	for _, c := range config.Inputs {
		i, err := input.New(c, b.Publisher, done, cp.States())
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, i)
	}

	bt := &Journalbeat{
		inputs:     inputs,
		done:       done,
		config:     config,
		pipeline:   b.Publisher,
		checkpoint: cp,
		logger:     logp.NewLogger("journalbeat"),
	}

	return bt, nil
}

// Run sets up the ACK handler and starts inputs to read and forward events to outputs.
func (bt *Journalbeat) Run(b *beat.Beat) error {
	bt.logger.Info("journalbeat is running! Hit CTRL-C to stop it.")
	defer bt.logger.Info("journalbeat is stopping")

	err := bt.pipeline.SetACKHandler(beat.PipelineACKHandler{
		ACKLastEvents: func(data []interface{}) {
			for _, datum := range data {
				if st, ok := datum.(checkpoint.JournalState); ok {
					bt.checkpoint.PersistState(st)
				}
			}
		},
	})
	if err != nil {
		return err
	}
	defer bt.checkpoint.Shutdown()

	var wg sync.WaitGroup
	for _, i := range bt.inputs {
		wg.Add(1)
		go bt.runInput(i, &wg)
	}

	wg.Wait()

	return nil
}

func (bt *Journalbeat) runInput(i *input.Input, wg *sync.WaitGroup) {
	defer wg.Done()
	i.Run()
}

// Stop stops the beat and its inputs.
func (bt *Journalbeat) Stop() {
	close(bt.done)
	for _, i := range bt.inputs {
		i.Stop()
	}
}
