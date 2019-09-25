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

package add_nomad_metadata

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/nomad"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
)

const (
	timeout = time.Second * 5
)

type nomadAnnotator struct {
	watcher  nomad.Watcher
	indexers *Indexers
	matchers *Matchers
	cache    *cache
}

func init() {
	processors.RegisterPlugin("add_nomad_metadata", New)
}

// New constructs a new add_nomad_metadata processor.
func New(cfg *common.Config) (processors.Processor, error) {
	config := defaultNomadAnnotatorConfig()

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("fail to unpack the nomad configuration: %s", err)
	}

	//Load default indexer configs
	if config.DefaultIndexers.Enabled == true {
		Indexing.RLock()
		for key, cfg := range Indexing.GetDefaultIndexerConfigs() {
			config.Indexers = append(config.Indexers, map[string]common.Config{key: cfg})
		}
		Indexing.RUnlock()
	}

	//Load default matcher configs
	if config.DefaultMatchers.Enabled == true {
		Indexing.RLock()
		for key, cfg := range Indexing.GetDefaultMatcherConfigs() {
			config.Matchers = append(config.Matchers, map[string]common.Config{key: cfg})
		}
		Indexing.RUnlock()
	}

	client, err := nomad.GetNomadClient()
	if err != nil {
		logp.Err("nomad: Couldn't create client")
		return nil, err
	}

	metaGen, err := nomad.NewMetaGenerator(cfg, client)
	if err != nil {
		return nil, err
	}

	indexers := NewIndexers(config.Indexers, metaGen)
	matchers := NewMatchers(config.Matchers)

	logp.Debug("nomad", "Using host: %s", config.Host)
	logp.Debug("nomad", "Initializing watcher")

	watcher, err := nomad.NewWatcher(client, nomad.WatchOptions{
		SyncTimeout: config.SyncPeriod,
		Node:        config.Host,
		Namespace:   config.Namespace,
	})

	if err != nil {
		logp.Err("ERROR creating Watcher %v", err.Error())
		return nil, err
	}

	processor := &nomadAnnotator{
		watcher:  watcher,
		indexers: indexers,
		matchers: matchers,
		cache:    newCache(config.CleanupTimeout),
	}

	watcher.AddEventHandler(nomad.ResourceEventHandlerFuncs{
		AddFunc: func(alloc nomad.Resource) {
			processor.addAllocation(&alloc)
		},
		DeleteFunc: func(alloc nomad.Resource) {
			processor.removeAllocation(&alloc)
		},
		UpdateFunc: func(alloc nomad.Resource) {
			processor.removeAllocation(&alloc)
			processor.addAllocation(&alloc)
		},
	})

	if err := watcher.Start(); err != nil {
		return nil, err
	}

	return processor, nil
}

func (n *nomadAnnotator) Run(event *beat.Event) (*beat.Event, error) {
	index := n.matchers.MetadataIndex(event.Fields)
	if index == "" {
		return event, nil
	}

	metadata := n.cache.get(index)
	if metadata == nil {
		return event, nil
	}

	event.Fields.DeepUpdate(common.MapStr{
		"nomad": metadata.Clone(),
	})

	return event, nil
}

func (n *nomadAnnotator) addAllocation(alloc *nomad.Resource) {
	metadata := n.indexers.GetMetadata(alloc)

	for _, m := range metadata {
		n.cache.set(m.Index, m.Data)
	}
}

func (n *nomadAnnotator) removeAllocation(alloc *nomad.Resource) {
	metadata := n.indexers.GetIndexes(alloc)

	for _, idx := range metadata {
		n.cache.delete(idx)
	}
}

func (n *nomadAnnotator) String() string {
	return "add_nomad_metadata"
}
