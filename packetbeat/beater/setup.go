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
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/packetbeat/config"
	"github.com/elastic/beats/v7/packetbeat/flows"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/sniffer"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func setupSniffer(cfg config.Config, protocols *protos.ProtocolsStruct, workerFactory sniffer.WorkerFactory) (*sniffer.Sniffer, error) {
	icmp, err := cfg.ICMP()
	if err != nil {
		return nil, err
	}

	filter := cfg.Interfaces.BpfFilter
	if filter == "" && !cfg.Flows.IsEnabled() {
		filter = protocols.BpfFilter(cfg.Interfaces.WithVlans, icmp.Enabled())
	}

	return sniffer.New(false, filter, workerFactory, cfg.Interfaces)
}

func setupFlows(pipeline beat.Pipeline, watcher procs.ProcessesWatcher, cfg config.Config) (*flows.Flows, error) {
	if !cfg.Flows.IsEnabled() {
		return nil, nil
	}

	processors, err := processors.New(cfg.Flows.Processors)
	if err != nil {
		return nil, err
	}

	clientConfig := beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			EventMetadata: cfg.Flows.EventMetadata,
			Processor:     processors,
			KeepNull:      cfg.Flows.KeepNull,
		},
	}
	if cfg.Flows.Index != "" {
		clientConfig.Processing.Meta = mapstr.M{"raw_index": cfg.Flows.Index}
	}

	client, err := pipeline.ConnectWith(clientConfig)
	if err != nil {
		return nil, err
	}

	return flows.NewFlows(client.PublishAll, watcher, cfg.Flows)
}
