// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pub

import (
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/add_data_stream"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/ecs"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	eventModule = "osquery_manager"
)

type Publisher struct {
	b   *beat.Beat
	log *logp.Logger

	mx     sync.Mutex
	client beat.Client
}

func New(b *beat.Beat, log *logp.Logger) *Publisher {
	return &Publisher{
		b:   b,
		log: log,
	}
}

func (p *Publisher) Configure(inputs []config.InputConfig) error {
	if len(inputs) == 0 {
		return nil
	}

	p.mx.Lock()
	defer p.mx.Unlock()

	processors, err := p.processorsForInputsConfig(inputs)
	if err != nil {
		return err
	}

	p.log.Debugf("Connect publisher with processors: %d", len(processors.All()))
	// Connect publisher
	client, err := p.b.Publisher.ConnectWith(beat.ClientConfig{
		Processing: beat.ProcessingConfig{
			Processor: processors,
		},
	})
	if err != nil {
		return err
	}

	// Swap client
	oldclient := p.client
	p.client = client
	if oldclient != nil {
		oldclient.Close()
	}
	return nil
}

func (p *Publisher) Publish(index, actionID, responseID string, meta map[string]interface{}, hits []map[string]interface{}, ecsm ecs.Mapping, reqData interface{}) {
	p.mx.Lock()
	defer p.mx.Unlock()

	for _, hit := range hits {
		event := hitToEvent(index, p.b.Info.Name, actionID, responseID, meta, hit, ecsm, reqData)
		p.client.Publish(event)
	}
	p.log.Infof("%d events sent to index %s", len(hits), index)
}

func (p *Publisher) Close() {
	p.mx.Lock()
	defer p.mx.Unlock()

	if p.client != nil {
		p.client.Close()
		p.client = nil
	}
}

func (p *Publisher) processorsForInputsConfig(inputs []config.InputConfig) (procs *processors.Processors, err error) {
	procs = processors.NewList(nil)

	// Use only first input processor
	// Every input will have a processor that adds the elastic_agent info, we need only one
	// Not expecting other processors at the moment and this needs to work for 7.13
	for _, input := range inputs {
		if len(input.Processors) > 0 {
			// Attach the data_stream processor. This will append the data_stream attributes to the events.
			// This is needed for the proper logstash auto-discovery of the destination datastream for the results.
			ds := add_data_stream.DataStream{
				Namespace: input.Datastream.Namespace,
				Dataset:   input.Datastream.Dataset,
				Type:      input.Datastream.Type,
			}
			if ds.Namespace == "" {
				ds.Namespace = config.DefaultNamespace
			}
			if ds.Dataset == "" {
				ds.Dataset = config.DefaultDataset
			}
			if ds.Type == "" {
				ds.Type = config.DefaultType
			}

			procs.AddProcessor(add_data_stream.New(ds))

			userProcs, err := processors.New(input.Processors)
			if err != nil {
				return nil, err
			}
			procs.AddProcessors(*userProcs)
			break
		}
	}
	return procs, nil
}

func hitToEvent(index, eventType, actionID, responseID string, meta, hit map[string]interface{}, ecsm ecs.Mapping, reqData interface{}) beat.Event {
	var fields mapstr.M

	if len(ecsm) > 0 {
		// Map ECS fields if the mapping is provided
		fields = mapstr.M(ecsm.Map(hit))
	} else {
		fields = mapstr.M{}
	}

	// Add event.module for ECS
	// There could be already "event" properties set, preserve them and set the "event.module"
	var evf map[string]interface{}
	ievf, ok := fields["event"]
	if ok {
		evf, ok = ievf.(map[string]interface{})
	}
	if !ok {
		evf = make(map[string]interface{})
	}
	evf["module"] = eventModule
	fields["event"] = evf

	fields["type"] = eventType
	fields["action_id"] = actionID
	fields["osquery"] = hit
	if meta != nil {
		fields["osquery_meta"] = meta
	}

	event := beat.Event{
		Timestamp: time.Now(),
		Fields:    fields,
	}

	if reqData != nil {
		event.Fields["action_data"] = reqData
	}

	if responseID != "" {
		event.Fields["response_id"] = responseID
	}
	if index != "" {
		event.Meta = mapstr.M{events.FieldMetaRawIndex: index}
	}

	return event
}
