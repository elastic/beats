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

package beater

import (
	"errors"

	conf "github.com/elastic/beats/v7/pkg/config"
)

const defaultRegistryFile = ".winlogbeat.yml"

type settings struct {
	// Filebeat is the translated config section to hand to the Filebeat beater.
	Filebeat *conf.C
	// EventLogs are the enabled event_logs entries, in order, untranslated.
	EventLogs []*conf.C
	// RegistryFile is the legacy state file path as configured (relative or
	// absolute), defaulting to defaultRegistryFile.
	RegistryFile string
	// OverwritePipelines mirrors winlogbeat.overwrite_pipelines.
	OverwritePipelines bool
}

// translateConfig turns the winlogbeat config section into the shape the
// Filebeat beater expects: each enabled event log becomes a winlog input,
// registry_flush becomes registry.flush and registry_file is dropped because
// the Filebeat beater rejects it. old is not modified.
func translateConfig(old *conf.C) (settings, error) {
	typed := struct {
		EventLogs          []*conf.C `config:"event_logs"`
		RegistryFile       string    `config:"registry_file"`
		OverwritePipelines bool      `config:"overwrite_pipelines"`
	}{RegistryFile: defaultRegistryFile}
	err := old.Unpack(&typed)
	if err != nil {
		return settings{}, err
	}

	var (
		eventLogs []*conf.C
		inputs    []any
	)
	for _, cfg := range typed.EventLogs {
		if !cfg.Enabled() {
			continue
		}
		var in map[string]any
		err = cfg.Unpack(&in)
		if err != nil {
			return settings{}, err
		}
		if in == nil {
			in = make(map[string]any)
		}
		in["type"] = "winlog"
		inputs = append(inputs, in)
		eventLogs = append(eventLogs, cfg)
	}
	if len(eventLogs) == 0 {
		return settings{}, errors.New("at least one event log must be configured as part of event_logs")
	}

	var raw map[string]any
	err = old.Unpack(&raw)
	if err != nil {
		return settings{}, err
	}
	delete(raw, "event_logs")
	delete(raw, "registry_file")
	raw["inputs"] = inputs
	flush, ok := raw["registry_flush"]
	if ok {
		delete(raw, "registry_flush")
		registry, _ := raw["registry"].(map[string]any)
		if registry == nil {
			registry = make(map[string]any)
		}
		registry["flush"] = flush
		raw["registry"] = registry
	}

	fb, err := conf.NewConfigFrom(raw)
	if err != nil {
		return settings{}, err
	}
	return settings{
		Filebeat:           fb,
		EventLogs:          eventLogs,
		RegistryFile:       typed.RegistryFile,
		OverwritePipelines: typed.OverwritePipelines,
	}, nil
}
