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
	"time"

	nomad "github.com/hashicorp/nomad/api"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	nomadlib "github.com/elastic/beats/libbeat/common/nomad"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/processors/actions"
	jsprocessor "github.com/elastic/beats/libbeat/processors/script/javascript/module/processor"
)

const (
	processorName        = "add_nomad_metadata"
	nomadRegexDataPrefix = "nomad."
	timeout              = time.Second * 5
)

type nomadAnnotator struct {
	log *logp.Logger

	sourceProcessor processors.Processor
	cache           *cache
	datacenter      string
	metaPrefix      string
}

func init() {
	processors.RegisterPlugin(processorName, New)
	jsprocessor.RegisterPlugin("AddNomadMetadata", New)
}

// New constructs a new add_kubernetes_metadata processor.
func New(cfg *common.Config) (processors.Processor, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrapf(err, "fail to unpack the %v configuration", processorName)
	}

	client, err := nomadlib.NewClient(config.Address, config.Region, config.SecretID, nil)
	if err != nil {
		logp.Err("nomad: couldn't create nomad client: %v", err)
		return nil, err
	}

	nodeID := config.Node
	if nodeID == "" {
		if nodeID, err = nomadlib.GetLocalNodeID(client); err != nil {
			logp.Err("nomad: couldn't get nomad node ID: %v", err)
			return nil, err
		}
	}
	node, _, err := client.Nodes().Info(nodeID, nil)
	if err != nil {
		logp.Err("nomad: couldn't get node information: %v", err)
		return nil, err
	}

	var procConf, _ = common.NewConfigFrom(map[string]interface{}{
		"field":  "log.file.path",
		"prefix": nomadRegexDataPrefix,
		"regexp": `.*/(?P<allocation>.+)/alloc/logs/(?P<task>.+)\.(?P<stream>std.+)\.[0-9]+`,
	})
	sourceProcessor, err := actions.NewExtractRegexp(procConf)
	if err != nil {
		return nil, err
	}

	annotator := &nomadAnnotator{
		log:             logp.NewLogger(processorName),
		sourceProcessor: sourceProcessor,
		cache:           newCache(config.CleanupTimeout),
		datacenter:      node.Datacenter,
		metaPrefix:      config.MetaPrefix,
	}

	watcher, err := nomadlib.NewWatcherWithClient(nomadlib.WrapClient(client), nodeID, annotator.modifiedAllocation)
	if err != nil {
		logp.Err("nomad: couldn't create allocation watcher")
		return nil, err
	}

	go watcher.Start()

	return annotator, nil
}

func (n *nomadAnnotator) Run(event *beat.Event) (*beat.Event, error) {
	var err error
	if lfp, _ := event.Fields.GetValue("log.file.path"); lfp != nil { // TODO: what if lfp is nil
		if event, err = n.sourceProcessor.Run(event); err != nil {
			n.log.Debugf("error while extracting container ID from source path: %v", err)
			return event, nil
		}
	}

	allocID, err := event.GetValue(nomadRegexDataPrefix + "allocation")
	if err != nil {
		n.log.Debugf("error while extracting allocation ID from event: %v", err)
		return event, nil
	}

	properties := n.cache.get(allocID.(string))
	if properties == nil {
		return event, nil
	}

	task, err := event.GetValue(nomadRegexDataPrefix + "task")
	if err != nil {
		n.log.Debugf("error while extracting task from event: %v", err)
		return event, nil
	}

	metadata := n.cache.get(allocID.(string) + task.(string))
	if metadata == nil {
		return event, nil
	}

	event.Fields.DeepUpdate(common.MapStr{
		"nomad": properties.Clone(),
	})

	event.PutValue("nomad.meta", metadata.Clone())
	event.PutValue("nomad.dc", n.datacenter)
	return event, nil
}

func (n *nomadAnnotator) modifiedAllocation(alloc *nomad.Allocation) {
	if nomadlib.IsTerminal(alloc) {
		n.cache.delete(alloc.ID)
		for k := range alloc.TaskStates {
			n.cache.delete(alloc.ID + k)
		}
		return
	}
	n.cache.set(alloc.ID, nomadlib.FetchProperties(alloc))
	for k := range alloc.TaskStates {
		n.cache.set(alloc.ID+k, nomadlib.FetchMetadata(alloc, k, n.metaPrefix))
	}
}

func (*nomadAnnotator) String() string {
	return "add_nomad_metadata"
}
