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

package input

import (
	"fmt"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/feature"
)

type Context struct {
	States        []file.State
	Done          chan struct{}
	BeatDone      chan struct{}
	DynamicFields *common.MapStrPointer
	Meta          map[string]string
}

// Factory is used to register functions creating new Input instances.
type Factory = func(config *common.Config, connector channel.Connector, context Context) (Input, error)

var namespace = "filebeat.input"

// Register registers a new feature with the registry.
func Register(name string, factory Factory) error {
	return feature.Register(Feature(name, factory, feature.NewDetails(name, "", feature.Undefined)))
}

// GetFactory find an input type factory if available.
func GetFactory(name string) (Factory, error) {
	f, err := feature.Registry.Lookup(namespace, name)
	if err != nil {
		return nil, err
	}

	factory, ok := f.Factory().(Factory)
	if !ok {
		return nil, fmt.Errorf("invalid input type, received: %T", f.Factory())
	}

	return factory, nil
}

// Feature return a new input feature.
func Feature(name string, factory Factory, description feature.Describer) *feature.Feature {
	return feature.New(namespace, name, factory, description)
}
