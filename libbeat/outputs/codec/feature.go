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

package codec

import (
	"fmt"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/feature"
)

// Namespace exposes the namespace of the codec to the registry.
var Namespace = "libbeat.output.codec"

// Factory is the function type returned by the global registry.
type Factory func(beat.Info, *common.Config) (Codec, error)

// Config codec configuration.
type Config struct {
	Namespace common.ConfigNamespace `config:",inline"`
}

// Plugin accepts a codec to be registered as a plugin.
func Plugin(name string, f Factory) *feature.Feature {
	return Feature(name, f, feature.NewDetails(name, "", feature.Undefined))
}

// FindFactory find, assert and return the factory to create a specific codec.
func FindFactory(name string) (Factory, error) {
	f, err := feature.Registry.Lookup(Namespace, name)
	if err != nil {
		return nil, err
	}

	factory, ok := f.Factory().(Factory)
	if !ok {
		return nil, fmt.Errorf("invalid codec type, received: %T", f.Factory())
	}
	return factory, nil
}

// CreateEncoder creates a new encoder from the provided configuration.
func CreateEncoder(info beat.Info, cfg Config) (Codec, error) {
	// default to json codec
	codec := "json"
	if name := cfg.Namespace.Name(); name != "" {
		codec = name
	}

	factory, err := FindFactory(codec)
	if err != nil {
		return nil, err
	}
	return factory(info, cfg.Namespace.Config())
}

// Feature creates a new codec.
func Feature(name string, f Factory, description feature.Describer) *feature.Feature {
	return feature.New(Namespace, name, f, description)
}
