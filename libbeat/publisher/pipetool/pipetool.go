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

package pipetool

import (
	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/acker"
)

// connectEditPipeline modifies the client configuration using edit before calling
// edit.
type connectEditPipeline struct {
	parent beat.PipelineConnector
	edit   ConfigEditor
}

// ConfigEditor modifies the client configuration before connecting to a Pipeline.
type ConfigEditor func(beat.ClientConfig) (beat.ClientConfig, error)

func (p *connectEditPipeline) Connect() (beat.Client, error) {
	return p.ConnectWith(beat.ClientConfig{})
}

func (p *connectEditPipeline) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	cfg, err := p.edit(cfg)
	if err != nil {
		return nil, err
	}
	return p.parent.ConnectWith(cfg)
}

// wrapClientPipeline applies edit to the beat.Client returned by Connect and ConnectWith.
// The edit function can wrap the client to add additional functionality to clients
// that connect to the pipeline.
type wrapClientPipeline struct {
	parent  beat.PipelineConnector
	wrapper ClientWrapper
}

// ClientWrapper allows client instances to be wrapped.
type ClientWrapper func(beat.Client) beat.Client

func (p *wrapClientPipeline) Connect() (beat.Client, error) {
	return p.ConnectWith(beat.ClientConfig{})
}

func (p *wrapClientPipeline) ConnectWith(cfg beat.ClientConfig) (beat.Client, error) {
	client, err := p.parent.ConnectWith(cfg)
	if err == nil {
		client = p.wrapper(client)
	}
	return client, err
}

// WithClientConfigEdit creates a pipeline connector, that allows the
// beat.ClientConfig to be modified before connecting to the underlying
// pipeline.
// The edit function is applied before calling Connect or ConnectWith.
func WithClientConfigEdit(pipeline beat.PipelineConnector, edit ConfigEditor) beat.PipelineConnector {
	return &connectEditPipeline{parent: pipeline, edit: edit}
}

// WithDefaultGuarantee sets the default sending guarantee to `mode` if the
// beat.ClientConfig does not set the mode explicitly.
func WithDefaultGuarantees(pipeline beat.PipelineConnector, mode beat.PublishMode) beat.PipelineConnector {
	return WithClientConfigEdit(pipeline, func(cfg beat.ClientConfig) (beat.ClientConfig, error) {
		if cfg.PublishMode == beat.DefaultGuarantees {
			cfg.PublishMode = mode
		}
		return cfg, nil
	})
}

func WithACKer(pipeline beat.PipelineConnector, a beat.ACKer) beat.PipelineConnector {
	return WithClientConfigEdit(pipeline, func(cfg beat.ClientConfig) (beat.ClientConfig, error) {
		if h := cfg.ACKHandler; h != nil {
			cfg.ACKHandler = acker.Combine(a, h)
		} else {
			cfg.ACKHandler = a
		}
		return cfg, nil
	})
}

// WithClientWrapper calls wrap on beat.Client instance, after a successful
// call to `pipeline.Connect` or `pipeline.ConnectWith`. The wrap function can
// wrap the client to provide additional functionality.
func WithClientWrapper(pipeline beat.PipelineConnector, wrap ClientWrapper) beat.PipelineConnector {
	return &wrapClientPipeline{parent: pipeline, wrapper: wrap}
}

// WithDynamicFields ensures that dynamicFields from autodiscovery are setup
// when connecting to the publisher pipeline.
// Processing.DynamicFields will only be overwritten if not is not already set.
func WithDynamicFields(pipeline beat.PipelineConnector, dynamicFields *common.MapStrPointer) beat.PipelineConnector {
	if dynamicFields == nil {
		return pipeline
	}

	return WithClientConfigEdit(pipeline, func(cfg beat.ClientConfig) (beat.ClientConfig, error) {
		if cfg.Processing.DynamicFields == nil {
			cfg.Processing.DynamicFields = dynamicFields
		}
		return cfg, nil
	})
}
