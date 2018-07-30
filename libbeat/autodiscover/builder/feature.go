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

package builder

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/feature"
)

// Namespace is the registry namespace for autodiscover builders.
var Namespace = "libbeat.autodiscover.builder"

// Builder provides an interface by which configs can be built from provider metadata
type Builder interface {
	// CreateConfig creates a config from hints passed from providers
	CreateConfig(event bus.Event) []*common.Config
}

// Config settings
type Config struct {
	Type string `config:"type"`
}

// Plugin accepts a builder to be registered as a plugin
func Plugin(name string, factory Factory) *feature.Feature {
	return Feature(name, factory, feature.NewDetails(name, "", feature.Undefined))
}

// Feature defines a new builder feature.
func Feature(name string, factory Factory, description feature.Describer) *feature.Feature {
	return feature.New(Namespace, name, factory, description)
}

// Factory is a func used to generate a Builder object
type Factory func(*common.Config) (Builder, error)

// FindFactory returns the builder with the giving name or return an error.
func FindFactory(name string) (Factory, error) {
	f, err := feature.Registry.Lookup(Namespace, name)
	if err != nil {
		return nil, err
	}

	factory, ok := f.Factory().(Factory)
	if !ok {
		return nil, fmt.Errorf("incompatible type for builder, received: '%T'", f.Factory())
	}

	return factory, nil
}

// Build reads provider configuration and instatiate one.
func Build(c *common.Config) (Builder, error) {
	var config Config
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	builder, err := FindFactory(config.Type)
	if err != nil {
		return nil, err
	}

	return builder(c)
}
