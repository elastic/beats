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

package appenders

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/libbeat/processors"
)

// Namespace is the registry namespace for autodiscover appenders.
var Namespace = "libbeat.autodiscover.appender"

// Appender provides an interface by which extra configuration can be added into configs
type Appender interface {
	// Append takes a processed event and add extra configuration
	Append(event bus.Event)
}

// Config settings
type Config struct {
	Type            string                      `config:"type"`
	ConditionConfig *processors.ConditionConfig `config:"condition"`
}

// Feature defines a new appender feature.
func Feature(name string, factory Factory, stability feature.Stability) *feature.Feature {
	return feature.New(Namespace, name, factory, stability)
}

// Factory is a func used to generate a Appender object
type Factory func(*common.Config) (Appender, error)

// FindFactory returns the appender with the giving name or return an error.
func FindFactory(name string) (Factory, error) {
	f, err := feature.Registry.Lookup(Namespace, name)
	if err != nil {
		return nil, err
	}

	factory, ok := f.Factory().(Factory)
	if !ok {
		return nil, fmt.Errorf("incompatible type for appender, received: '%T'", f.Factory())
	}

	return factory, nil
}

// Build reads provider configuration and instantiate one
func Build(c *common.Config) (Appender, error) {
	var config Config
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}

	appender, err := FindFactory(config.Type)
	if err != nil {
		return nil, err
	}

	return appender(c)
}
