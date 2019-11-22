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
	"github.com/elastic/beats/libbeat/common/fmtstr"
	"github.com/elastic/beats/libbeat/processors"
)

type OutletFactory struct {
	done <-chan struct{}

	eventer  beat.ClientEventer
	wgEvents eventCounter
	beatInfo beat.Info
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
	KeepNull             bool                    `config:"keep_null"`

	// implicit event fields
	Type        string `config:"type"`         // input.type
	ServiceType string `config:"service.type"` // service.type

	// hidden filebeat modules settings
	Module  string `config:"_module_name"`  // hidden setting
	Fileset string `config:"_fileset_name"` // hidden setting

	// Output meta data settings
	Pipeline string                   `config:"pipeline"` // ES Ingest pipeline name
	Index    fmtstr.EventFormatString `config:"index"`    // ES output index pattern
}

// NewOutletFactory creates a new outlet factory for
// connecting an input to the publisher pipeline.
func NewOutletFactory(
	done <-chan struct{},
	wgEvents eventCounter,
	beatInfo beat.Info,
) *OutletFactory {
	o := &OutletFactory{
		done:     done,
		wgEvents: wgEvents,
		beatInfo: beatInfo,
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
func (f *OutletFactory) Create(p beat.Pipeline) Connector {
	return &pipelineConnector{parent: f, pipeline: p}
}

func (e *clientEventer) Closing()                        {}
func (e *clientEventer) Closed()                         {}
func (e *clientEventer) Published()                      {}
func (e *clientEventer) FilteredOut(evt beat.Event)      {}
func (e *clientEventer) DroppedOnPublish(evt beat.Event) { e.wgEvents.Done() }
