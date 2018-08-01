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

package monitors

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/feature"
)

// Type of Monitor used by Heartbeat, currently only supporting ActiveMonitor.
//go:generate stringer -type=Type -linecomment=true
type Type uint8

// Namespace return the registry namespace for the current monitor type.
func (t Type) Namespace() string {
	return fmt.Sprintf("heartbeat.%s", strings.ToLower(t.String()))
}

// Type of supported monitor.
const (
	ActiveMonitor Type = iota + 1
)

type entry struct {
	info    Info
	builder ActiveBuilder
}

// Info contains the generatl information about a monitor.
type Info struct {
	Name string
	Type Type
}

// Job interface is the interface that a monitor need to implements to be executed by the scheduler.
type Job interface {
	Name() string
	Run() (beat.Event, []JobRunner, error)
}

type JobRunner func() (beat.Event, []JobRunner, error)

type TaskRunner interface {
	Run() (common.MapStr, []TaskRunner, error)
}

// ActiveBuilder is the factory signature to create a new active monitor.
type ActiveBuilder func(Info, *common.Config) ([]Job, error)

// Factory is the type returning by the find factory linking the ActiveBuilder and returning the jobs
// that need to be executed by the scheduler.
type Factory func(*common.Config) ([]Job, error)

// ActiveFeature creates a new Active monitor.
func ActiveFeature(name string, factory ActiveBuilder, description feature.Describer) *feature.Feature {
	entry := entry{
		info: Info{
			Name: name,
			Type: ActiveMonitor,
		},
		builder: factory,
	}
	return feature.New(ActiveMonitor.Namespace(), name, entry, description)
}

// RegisterActive is a backward compatible shim to make the old api work with the new global registry.
func RegisterActive(name string, builder ActiveBuilder) {
	f := ActiveFeature(name, builder, feature.NewDetails(name, "", feature.Undefined))
	feature.MustRegister(f)
}

// Registry wraps the global registry.
var Registry = newRegistrar(feature.Registry)

// Registrar wrapper around the registry.
type Registrar struct {
	registry *feature.FeatureRegistry
}

func newRegistrar(registry *feature.FeatureRegistry) *Registrar {
	return &Registrar{registry: registry}
}

// GetFactory return an monitor for the request type and name or an error.
func (r *Registrar) GetFactory(name string) (Factory, error) {
	e, err := r.getEntry(name)
	if err != nil {
		return nil, err
	}
	return e.Create, nil
}

// Query returns general information about a monitor.
func (r *Registrar) Query(name string) (*Info, error) {
	e, err := r.getEntry(name)
	if err != nil {
		return nil, err
	}
	return &e.info, nil
}

func (r *Registrar) getEntry(name string) (*entry, error) {
	f, err := r.registry.Lookup(ActiveMonitor.Namespace(), name)
	if err != nil {
		return nil, err
	}

	e, ok := f.Factory().(entry)
	if !ok {
		return nil, fmt.Errorf("incompatible type for %s, expect entry, received: %T", name, f)
	}

	return &e, nil
}

func (e *entry) Create(cfg *common.Config) ([]Job, error) {
	return e.builder(e.info, cfg)
}
