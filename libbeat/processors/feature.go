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

package processors

import (
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/feature"
)

// Namespace exposes the processor type.
var Namespace = "libbeat.processor"

type processorPlugin struct {
	name   string
	constr Constructor
}

// Constructor is the factory method to create a new processor.
type Constructor func(config *common.Config) (Processor, error)

// Feature define a new feature.
func Feature(name string, factory Constructor, stability feature.Stability) *feature.Feature {
	return feature.New(Namespace, name, factory, stability)
}

// Find returns the processor factory and wrap it into a NewConditonal.
func Find(name string) (Constructor, error) {
	f, err := feature.Registry.Lookup(Namespace, name)
	if err != nil {
		return nil, err
	}

	factory, ok := f.Factory().(Constructor)
	if !ok {
		return nil, fmt.Errorf("invalid processor type, received: %T", f.Factory())
	}

	return NewConditional(factory), nil
}
