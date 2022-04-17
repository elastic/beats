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
	"github.com/menderesk/beats/v7/libbeat/feature"
)

// Namespace is the feature namespace for queue definition.
const Namespace = "libbeat.queue"

// RegisterQueueType registers a new queue type.
func RegisterQueueType(name string, factory Factory, details feature.Details) {
	feature.MustRegister(feature.New(Namespace, name, factory, details))
}

// FindFactory retrieves a queue types constructor. Returns nil if queue type is unknown
func FindFactory(name string) Factory {
	f, err := feature.GlobalRegistry().Lookup(Namespace, name)
	if err != nil {
		return nil
	}
	factory, ok := f.Factory().(Factory)
	if !ok {
		return nil
	}

	return factory
}
