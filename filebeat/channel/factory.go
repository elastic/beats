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

package channel

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
)

type OutletFactory struct {
	done <-chan struct{}

	eventer  beat.ClientEventer
	wgEvents eventCounter
}

type eventCounter interface {
	Add(n int)
	Done()
}

// clientEventer adjusts wgEvents if events are dropped during shutdown.
type clientEventer struct {
	wgEvents eventCounter
}

// inputOutletConfig defines common input settings
// for the publisher pipeline.
type inputOutletConfig struct {
	// event processing
	common.EventMetadata `config:",inline"`      // Fields and tags to add to events.
	Processors           processors.PluginConfig `config:"processors"`

	// implicit event fields
	Type        string `config:"type"`         // input.type
	ServiceType string `config:"service.type"` // service.type

	// hidden filebeat modules settings
	Module  string `config:"_module_name"`  // hidden setting
	Fileset string `config:"_fileset_name"` // hidden setting

	// Output meta data settings
	Pipeline string `config:"pipeline"` // ES Ingest pipeline name

}

// NewOutletFactory creates a new outlet factory for
// connecting an input to the publisher pipeline.
func NewOutletFactory(
	done <-chan struct{},
	wgEvents eventCounter,
) *OutletFactory {
	o := &OutletFactory{
		done:     done,
		wgEvents: wgEvents,
	}

	if wgEvents != nil {
		o.eventer = &clientEventer{wgEvents}
	}

	return o
}

// Create builds a new Outleter, while applying common input settings.
// Inputs and all harvesters use the same pipeline client instance.
// This guarantees ordering between events as required by the registrar for
// file.State updates
func (f *OutletFactory) Create(p beat.Pipeline, cfg *common.Config, dynFields *common.MapStrPointer) (Outleter, error) {
	config := inputOutletConfig{}
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	processors, err := processors.New(config.Processors)
	if err != nil {
		return nil, err
	}

	setMeta := func(to common.MapStr, key, value string) {
		if value != "" {
			to[key] = value
		}
	}

	meta := common.MapStr{}
	setMeta(meta, "pipeline", config.Pipeline)

	fields := common.MapStr{}
	setMeta(fields, "module", config.Module)
	if config.Module != "" && config.Fileset != "" {
		setMeta(fields, "dataset", config.Module+"."+config.Fileset)
	}
	if len(fields) > 0 {
		fields = common.MapStr{
			"event": fields,
		}
	}
	if config.Fileset != "" {
		fields.Put("fileset.name", config.Fileset)
	}
	if config.ServiceType != "" {
		fields.Put("service.type", config.ServiceType)
	} else if config.Module != "" {
		fields.Put("service.type", config.Module)
	}
	if config.Type != "" {
		fields.Put("input.type", config.Type)
	}

	client, err := p.ConnectWith(beat.ClientConfig{
		PublishMode: beat.GuaranteedSend,
		Processing: beat.ProcessingConfig{
			EventMetadata: config.EventMetadata,
			DynamicFields: dynFields,
			Meta:          meta,
			Fields:        fields,
			Processor:     processors,
		},
		Events: f.eventer,
	})
	if err != nil {
		return nil, err
	}

	outlet := newOutlet(client, f.wgEvents)
	if f.done != nil {
		return CloseOnSignal(outlet, f.done), nil
	}
	return outlet, nil
}

func (*clientEventer) Closing()   {}
func (*clientEventer) Closed()    {}
func (*clientEventer) Published() {}

func (c *clientEventer) FilteredOut(_ beat.Event) {}
func (c *clientEventer) DroppedOnPublish(_ beat.Event) {
	c.wgEvents.Done()
}
