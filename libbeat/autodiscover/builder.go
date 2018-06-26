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

package autodiscover

import (
	"errors"

	"github.com/elastic/beats/libbeat/autodiscover/builder"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
)

// Builder provides an interface by which configs can be built from provider metadata
type Builder = builder.Builder

// Builders is a list of Builder objects
type Builders []Builder

// BuilderConstructor is a func used to generate a Builder object
type BuilderConstructor = builder.Factory

// GetConfig creates configs for all builders initialized.
func (b Builders) GetConfig(event bus.Event) []*common.Config {
	var configs []*common.Config

	for _, builder := range b {
		if config := builder.CreateConfig(event); config != nil {
			configs = append(configs, config...)
		}
	}

	return configs
}

// NewBuilders instances the given list of builders. If hintsEnabled is true it will
// just enable the hints builder
func NewBuilders(bConfigs []*common.Config, hintsEnabled bool) (Builders, error) {
	var builders Builders
	if hintsEnabled {
		if len(bConfigs) > 0 {
			return nil, errors.New("hints.enabled is incompatible with manually defining builders")
		}

		hints, err := common.NewConfigFrom(map[string]string{"type": "hints"})
		if err != nil {
			return nil, err
		}

		bConfigs = append(bConfigs, hints)
	}

	for _, bcfg := range bConfigs {
		b, err := builder.Build(bcfg)
		if err != nil {
			return nil, err
		}
		builders = append(builders, b)
	}

	return builders, nil
}
