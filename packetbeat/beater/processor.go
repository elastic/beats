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

	"github.com/mitchellh/hashstructure"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cfgfile"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/packetbeat/config"
	"github.com/elastic/beats/v7/packetbeat/flows"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/publish"
	"github.com/elastic/beats/v7/packetbeat/sniffer"
	conf "github.com/elastic/elastic-agent-libs/config"
)

type processor struct {
	wg              sync.WaitGroup
	publisher       *publish.TransactionPublisher
	flows           *flows.Flows
	sniffer         *sniffer.Sniffer
	shutdownTimeout time.Duration
	err             chan error
}

func newProcessor(shutdownTimeout time.Duration, publisher *publish.TransactionPublisher, flows *flows.Flows, sniffer *sniffer.Sniffer, err chan error) *processor {
	return &processor{
		publisher:       publisher,
		flows:           flows,
		sniffer:         sniffer,
		err:             err,
		shutdownTimeout: shutdownTimeout,
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
			p.err <- fmt.Errorf("sniffer loop failed: %w", err)
			return
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
	// wait for shutdownTimeout to let the publisher flush
	// whatever pending events
	if p.shutdownTimeout > 0 {
		time.Sleep(p.shutdownTimeout)
	}
	p.publisher.Stop()
}

// processorFactory controls construction of modules runners.
type processorFactory struct {
	name         string
	err          chan error
	beat         *beat.Beat
	configurator func(*conf.C) (config.Config, error)
}

func newProcessorFactory(name string, err chan error, beat *beat.Beat, configurator func(*conf.C) (config.Config, error)) *processorFactory {
	return &processorFactory{
		name:         name,
		err:          err,
		beat:         beat,
		configurator: configurator,
	}
}

// Create returns a new module runner that publishes to the provided pipeline, configured from cfg.
func (p *processorFactory) Create(pipeline beat.PipelineConnector, cfg *conf.C) (cfgfile.Runner, error) {
	config, err := p.configurator(cfg)
	if err != nil {
		logp.Err("Failed to read the beat config: %v, %v", err, config)
		return nil, err
	}
	id, err := configID(cfg)
	if err != nil {
		logp.Err("Failed to generate ID from config: %v, %v", err, config)
		return nil, err
	}
	if len(config.Interfaces) != 0 {
		// Install Npcap if needed. This needs to happen before any other
		// work on Windows, including config checking, because that involves
		// probing interfaces.
		//
		// Users may block installation of Npcap, so we defer the install
		// until we have a configuration that will tell us if it has been
		// blocked. To do this we must have a valid config.
		//
		// When Packetbeat is managed by fleet we will only have this if
		// Create has been called via the agent Reload process. We take
		// the opportunity to not install the DLL if there is no configured
		// interface.
		err := installNpcap(p.beat, cfg)
		if err != nil {
			return nil, err
		}
	}

	publisher, err := publish.NewTransactionPublisher(
		p.beat.Info.Name,
		p.beat.Publisher,
		config.IgnoreOutgoing,
		config.Interfaces[0].File == "",
		config.Interfaces[0].InternalNetworks,
	)
	if err != nil {
		return nil, err
	}

	var watch procs.ProcessesWatcher
	// Enable the process watcher only if capturing live traffic
	if config.Interfaces[0].File == "" {
		err = watch.Init(config.Procs)
		if err != nil {
			logp.Critical(err.Error())
			return nil, err
		}
	} else {
		logp.Info("Process watcher disabled when file input is used")
	}

	flows, err := setupFlows(pipeline, &watch, config)
	if err != nil {
		return nil, err
	}
	sniffer, err := setupSniffer(id, config, publisher, &watch, flows)
	if err != nil {
		return nil, err
	}

	return newProcessor(config.ShutdownTimeout, publisher, flows, sniffer, p.err), nil
}

// setupFlows returns a *flows.Flows that will publish to the provided pipeline,
// configured with cfg and process enrichment via the provided watcher.
func setupFlows(pipeline beat.Pipeline, watch *procs.ProcessesWatcher, cfg config.Config) (*flows.Flows, error) {
	if !cfg.Flows.IsEnabled() {
		return nil, nil
	}

	processors, err := processors.New(cfg.Flows.Processors)
	if err != nil {
		return nil, err
	}

	var meta mapstr.M
	if cfg.Flows.Index != "" {
		meta = mapstr.M{"raw_index": cfg.Flows.Index}
	}
	client, err := pipeline.ConnectWith(beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			EventMetadata: cfg.Flows.EventMetadata,
			Processor:     processors,
			KeepNull:      cfg.Flows.KeepNull,
			Meta:          meta,
		},
	})
	if err != nil {
		return nil, err
	}

	return flows.NewFlows(client.PublishAll, watch, cfg.Flows)
}

func setupSniffer(id string, cfg config.Config, pub *publish.TransactionPublisher, watch *procs.ProcessesWatcher, flows *flows.Flows) (*sniffer.Sniffer, error) {
	icmp, err := cfg.ICMP()
	if err != nil {
		return nil, err
	}

	// Ensure interfaces are uniquely represented so we don't listen on the
	// same interface with multiple sniffers.
	interfaces := make([]config.InterfaceConfig, 0, len(cfg.Interfaces))
	seen := make(map[uint64]bool)
	for _, iface := range cfg.Interfaces {
		// Currently we hash on all fields in the config. We can revise this in future.
		h, err := hashstructure.Hash(iface, nil)
		if err != nil {
			return nil, fmt.Errorf("could not deduplicate interface configurations: %w", err)
		}
		if seen[h] {
			continue
		}
		seen[h] = true
		interfaces = append(interfaces, iface)
	}

	logp.Debug("main", "Initializing protocol plugins")
	decoders := make(map[string]sniffer.Decoders)
	for i, iface := range interfaces {
		protocols := protos.NewProtocols()
		err = protocols.InitFiltered(false, iface.Device, pub, watch, cfg.Protocols, cfg.ProtocolsList)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize protocol analyzers for %s: %w", iface.Device, err)
		}
		decoders[iface.Device] = sniffer.DecodersFor(id, pub, protocols, watch, flows, cfg)
		if iface.BpfFilter != "" || cfg.Flows.IsEnabled() {
			continue
		}
		interfaces[i].BpfFilter = protocols.BpfFilter(iface.WithVlans, icmp.Enabled())
	}

	return sniffer.New(id, false, "", decoders, interfaces)
}

// CheckConfig performs a dry-run creation of a Packetbeat pipeline based
// on the provided configuration. This will involve setting up some dummy
// sniffers and so will need libpcap to be loaded.
func (p *processorFactory) CheckConfig(config *conf.C) error {
	runner, err := p.Create(pipeline.NewNilPipeline(), config)
	if err != nil {
		return err
	}
	runner.Stop()
	return nil
}

func configID(config *conf.C) (string, error) {
	var tmp struct {
		ID string `config:"id"`
	}
	if err := config.Unpack(&tmp); err != nil {
		return "", fmt.Errorf("error extracting ID: %w", err)
	}
	if tmp.ID != "" {
		return tmp.ID, nil
	}

	var h map[string]interface{}
	_ = config.Unpack(&h)
	id, err := hashstructure.Hash(h, nil)
	if err != nil {
		return "", fmt.Errorf("can not compute ID from configuration: %w", err)
	}

	return fmt.Sprintf("%16X", id), nil
}
