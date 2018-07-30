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
	"sort"
	"strings"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

type Factory func(*common.Config) ([]Job, error)

type ActiveBuilder func(Info, *common.Config) ([]Job, error)

type Job interface {
	Name() string
	Run() (beat.Event, []JobRunner, error)
}

type JobRunner func() (beat.Event, []JobRunner, error)

type TaskRunner interface {
	Run() (common.MapStr, []TaskRunner, error)
}

type Type uint8

type Info struct {
	Name string
	Type Type
}

const (
	ActiveMonitor Type = iota + 1
	PassiveMonitor
)

var Registry = newRegistrar()

type Registrar struct {
	modules map[string]entry
}

type entry struct {
	info    Info
	builder ActiveBuilder
}

func newRegistrar() *Registrar {
	return &Registrar{
		modules: map[string]entry{},
	}
}

func RegisterActive(name string, builder ActiveBuilder) {
	if err := Registry.AddActive(name, builder); err != nil {
		panic(err)
	}
}

func (r *Registrar) Register(name string, t Type, builder ActiveBuilder) error {
	if _, found := r.modules[name]; found {
		return fmt.Errorf("monitor type %v already exists", name)
	}

	info := Info{Name: name, Type: t}
	r.modules[name] = entry{info: info, builder: builder}

	return nil
}

func (r *Registrar) Query(name string) (Info, bool) {
	e, found := r.modules[name]
	return e.info, found
}

func (r *Registrar) GetFactory(name string) Factory {
	e, found := r.modules[name]
	if !found {
		return nil
	}
	return e.Create
}

func (r *Registrar) AddActive(name string, builder ActiveBuilder) error {
	return r.Register(name, ActiveMonitor, builder)
}

func (r *Registrar) String() string {
	var monitors []string
	for m := range r.modules {
		monitors = append(monitors, m)
	}
	sort.Strings(monitors)

	return fmt.Sprintf("Registry, monitors: %v",
		strings.Join(monitors, ", "))
}

func (e *entry) Create(cfg *common.Config) ([]Job, error) {
	return e.builder(e.info, cfg)
}

func (t Type) String() string {
	switch t {
	case ActiveMonitor:
		return "active"
	case PassiveMonitor:
		return "passive"
	default:
		return "unknown type"
	}
}
