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

package plugin

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/elastic/beats/v7/heartbeat/hbregistry"
	"github.com/elastic/beats/v7/heartbeat/monitors/jobs"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/plugin"
)

type PluginFactory struct {
	Name    string
	Aliases []string
	Builder PluginFactoryCreate
	Stats   RegistryRecorder
}

type PluginFactoryCreate func(string, *common.Config) (p Plugin, err error)

type Plugin struct {
	Jobs      []jobs.Job
	Close     func() error
	Endpoints int
}

var pluginKey = "heartbeat.monitor"

// stateGlobalRecorder records statistics across all plugin types
var stateGlobalRecorder = newRootGaugeRecorder(hbregistry.TelemetryRegistry)

func statsForPlugin(pluginName string) RegistryRecorder {
	return MultiRegistryRecorder{
		recorders: []RegistryRecorder{
			// state (telemetry)
			newPluginGaugeRecorder(pluginName, hbregistry.TelemetryRegistry),
			// Record global monitors / endpoints count
			NewPluginCountersRecorder(pluginName, hbregistry.StatsRegistry),
			// When stats for this plugin are updated, update the global stats as well
			stateGlobalRecorder,
		},
	}
}

func init() {
	plugin.MustRegisterLoader(pluginKey, func(ifc interface{}) error {
		p, ok := ifc.(PluginFactory)
		if !ok {
			return errors.New("plugin does not match monitor plugin type")
		}

		stats := statsForPlugin(p.Name)
		return GlobalPluginsReg.Register(PluginFactory{p.Name, p.Aliases, p.Builder, stats})
	})
}

// Type represents whether a plugin is active or passive.
type Type uint8

const (
	// ActiveMonitor represents monitorStarts that reach across the network to do things.
	ActiveMonitor Type = iota + 1
	// PassiveMonitor represents monitorStarts that receive inbound data.
	PassiveMonitor
)

// globalPluginsReg maintains the canonical list of valid Heartbeat monitorStarts at runtime.
var GlobalPluginsReg = NewPluginsReg()

type PluginsReg struct {
	monitors map[string]PluginFactory
}

func NewPluginsReg() *PluginsReg {
	return &PluginsReg{
		monitors: map[string]PluginFactory{},
	}
}

// Register registers a new active (as opposed to passive) monitor.
func Register(name string, builder PluginFactoryCreate, aliases ...string) {
	stats := statsForPlugin(name)
	if err := GlobalPluginsReg.Add(PluginFactory{name, aliases, builder, stats}); err != nil {
		panic(err)
	}
}

// ErrPluginAlreadyExists is returned when there is an attempt to register two plugins
// with the same pluginName.
type ErrPluginAlreadyExists PluginFactory

func (m ErrPluginAlreadyExists) Error() string {
	return fmt.Sprintf("monitor plugin named '%s' with Aliases %v already exists", m.Name, m.Aliases)
}

func (r *PluginsReg) Add(plugin PluginFactory) error {
	if _, exists := r.monitors[plugin.Name]; exists {
		return ErrPluginAlreadyExists(plugin)
	}
	r.monitors[plugin.Name] = plugin
	for _, alias := range plugin.Aliases {
		if _, exists := r.monitors[alias]; exists {
			return ErrPluginAlreadyExists(plugin)
		}
		r.monitors[alias] = plugin
	}
	return nil
}

func (r *PluginsReg) Register(plugin PluginFactory) error {
	if _, found := r.monitors[plugin.Name]; found {
		return fmt.Errorf("monitor type %v already exists", plugin.Name)
	}

	r.monitors[plugin.Name] = plugin

	return nil
}

func (r *PluginsReg) Get(name string) (PluginFactory, bool) {
	e, found := r.monitors[name]
	return e, found
}

func (r *PluginsReg) String() string {
	var monitors []string
	for m := range r.monitors {
		monitors = append(monitors, m)
	}
	sort.Strings(monitors)

	return fmt.Sprintf("globalPluginsReg, monitor: %v",
		strings.Join(monitors, ", "))
}
func (r *PluginsReg) MonitorNames() []string {
	names := make([]string, 0, len(r.monitors))
	for k := range r.monitors {
		names = append(names, k)
	}
	return names
}

func (e *PluginFactory) Create(cfg *common.Config) (p Plugin, err error) {
	return e.Builder(e.Name, cfg)
}
