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

package outputs

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/elastic-agent-libs/config"
)

var outputReg = map[string]Factory{}

// Factory is used by output plugins to build an output instance
type Factory func(
	im IndexManager,
	beat beat.Info,
	stats Observer,
	cfg *config.C) (Group, error)

// IndexManager provides additional index related services to the outputs.
type IndexManager interface {
	// BuildSelector can be used by an output to create an IndexSelector based on
	// the outputs configuration.
	// The defaultIndex is interpreted as format string and used as default fallback
	// if no index is configured or all indices are guarded using conditionals.
	BuildSelector(cfg *config.C) (IndexSelector, error)
}

// IndexSelector is used to find the index name an event shall be indexed to.
type IndexSelector interface {
	Select(event *beat.Event) (string, error)
}

// Group configures and combines multiple clients into load-balanced group of clients
// being managed by the publisher pipeline.
// If QueueFactory is set then the pipeline will use it to create the queue.
// Currently it is only used to activate the proxy queue when using the Shipper
// output, but it also provides a natural migration path for moving queue
// configuration into the outputs.
type Group struct {
	Clients        []Client
	BatchSize      int
	Retry          int
	QueueFactory   queue.QueueFactory
	EncoderFactory queue.EncoderFactory
}

// RegisterType registers a new output type.
func RegisterType(name string, f Factory) {
	if outputReg[name] != nil {
		panic(fmt.Errorf("output type  '%v' exists already", name))
	}
	outputReg[name] = f
}

// FindFactory finds an output type its factory if available.
func FindFactory(name string) Factory {
	return outputReg[name]
}

// Load creates and configures a output Group using a configuration object..
func Load(
	im IndexManager,
	info beat.Info,
	stats Observer,
	name string,
	config *config.C,
) (Group, error) {
	factory := FindFactory(name)
	if factory == nil {
		return Group{}, fmt.Errorf("output type %v undefined", name)
	}

	if stats == nil {
		stats = NewNilObserver()
	}
	return factory(im, info, stats, config)
}
