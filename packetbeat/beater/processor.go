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

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"

	"github.com/elastic/beats/v7/packetbeat/flows"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/publish"
	"github.com/elastic/beats/v7/packetbeat/sniffer"
)

type processor struct {
	wg      sync.WaitGroup
	flows   *flows.Flows
	sniffer *sniffer.Sniffer
	err     chan error
}

func newProcessor(flows *flows.Flows, sniffer *sniffer.Sniffer, err chan error) *processor {
	return &processor{
		flows:   flows,
		sniffer: sniffer,
		err:     err,
	}
}

func (p *processor) String() string {
	return "packetbeat.processor"
}

func (p *processor) Start() {
	if p.flows != nil {
		p.flows.Start()
	}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		err := p.sniffer.Run()
		if err != nil {
			p.err <- fmt.Errorf("Sniffer loop failed: %v", err)
		}
		p.err <- nil
	}()
}

func (p *processor) Stop() {
	p.sniffer.Stop()
	if p.flows != nil {
		p.flows.Stop()
	}
	p.wg.Wait()
}

type processorFactory struct {
	name      string
	err       chan error
	publisher *publish.TransactionPublisher
}

func newProcessorFactory(name string, err chan error, publisher *publish.TransactionPublisher) *processorFactory {
	return &processorFactory{
		name:      name,
		err:       err,
		publisher: publisher,
	}
}

func (p *processorFactory) Create(pipeline beat.PipelineConnector, cfg *common.Config) (cfgfile.Runner, error) {
	config := initialConfig()
	err := cfg.Unpack(&config)
	if err != nil {
		logp.Err("fails to read the beat config: %v, %v", err, config)
		return nil, err
	}

	// normalize agent-based configuration
	config, err = config.Normalize()
	if err != nil {
		logp.Err("failed to normalize the beat config: %v, %v", err, config)
		return nil, err
	}

	watcher := procs.ProcessesWatcher{}
	// Enable the process watcher only if capturing live traffic
	if config.Interfaces.File == "" {
		err = watcher.Init(config.Procs)
		if err != nil {
			logp.Critical(err.Error())
			return nil, err
		}
	} else {
		logp.Info("Process watcher disabled when file input is used")
	}

	logp.Debug("main", "Initializing protocol plugins")
	protocols := protos.NewProtocols()
	err = protocols.Init(false, p.publisher, watcher, config.Protocols, config.ProtocolsList)
	if err != nil {
		return nil, fmt.Errorf("Initializing protocol analyzers failed: %v", err)
	}
	flows, err := setupFlows(pipeline, watcher, config)
	if err != nil {
		return nil, err
	}
	sniffer, err := setupSniffer(config, protocols, workerFactory(p.publisher, protocols, watcher, flows, config))
	if err != nil {
		return nil, err
	}

	return newProcessor(flows, sniffer, p.err), nil
}

func (p *processorFactory) CheckConfig(config *common.Config) error {
	_, err := p.Create(pipeline.NewNilPipeline(), config)
	return err
}
