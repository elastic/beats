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

package queue

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/feature"
)

// Namespace is the feature namespace for queue definition.
var Namespace = "libbeat.queue"

// Global queue type registry for configuring and loading a queue instance
// via common.Config
var queueReg = map[string]Factory{}

// RegisterType registers a new queue type.
func RegisterType(name string, fn Factory) {
	f := feature.New(Namespace, name, fn, feature.Undefined)
	feature.MustRegister(f)
}

// FindFactory retrieves a queue types constructor. Returns nil if queue type is unknown
func FindFactory(name string) Factory {
	f, err := feature.Registry.Lookup(Namespace, name)
	if err != nil {
		return nil
	}
	factory, ok := f.Factory().(Factory)
	if !ok {
		return nil
	}

	return factory
}

// Load instantiates a new queue.
func Load(eventer Eventer, config common.ConfigNamespace) (Queue, error) {
	t, cfg := config.Name(), config.Config()
	if t == "" {
		t = "mem"
	}

	factory := FindFactory(t)
	if factory == nil {
		return nil, fmt.Errorf("queue type %v undefined", t)
	}
	return factory(eventer, cfg)
}

// Feature creates a new type of queue.
func Feature(name string, factory Factory, stability feature.Stability) *feature.Feature {
	return feature.New(Namespace, name, factory, stability)
}
