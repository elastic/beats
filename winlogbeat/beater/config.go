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

// neverCleanInactive is the clean_inactive set on each input unless the user
// sets one. The cursor input has no "never" value (zero or less removes a
// position as soon as its reader stops), and old Winlogbeat kept positions
// forever, so use about 100 years.
const neverCleanInactive = "876000h"

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
		RegistryFlush      any       `config:"registry_flush"`
		OverwritePipelines bool      `config:"overwrite_pipelines"`
	}{RegistryFile: defaultRegistryFile}
	err := old.Unpack(&typed)
	if err != nil {
		return settings{}, err
	}

	// Copy and edit config values rather than unpacking and re-parsing
	// them, so that escaped variables ($${...}) are not expanded again.
	fb := conf.NewConfig()
	err = fb.Merge(old)
	if err != nil {
		return settings{}, err
	}
	for _, key := range []string{"event_logs", "registry_file", "registry_flush"} {
		_, err = fb.Remove(key, -1)
		if err != nil {
			return settings{}, err
		}
	}

	var eventLogs []*conf.C
	for _, cfg := range typed.EventLogs {
		if !cfg.Enabled() {
			continue
		}
		in := conf.NewConfig()
		err = in.Merge(cfg)
		if err != nil {
			return settings{}, err
		}
		err = in.SetString("type", -1, "winlog")
		if err != nil {
			return settings{}, err
		}
		if !in.HasField("clean_inactive") {
			err = in.SetString("clean_inactive", -1, neverCleanInactive)
			if err != nil {
				return settings{}, err
			}
		}
		err = fb.SetChild("inputs", len(eventLogs), in)
		if err != nil {
			return settings{}, err
		}
		eventLogs = append(eventLogs, cfg)
	}
	if len(eventLogs) == 0 {
		return settings{}, errors.New("at least one event log must be configured as part of event_logs")
	}

	if typed.RegistryFlush != nil {
		// A flush interval is a duration, so it has no variables to expand.
		err = fb.Merge(map[string]any{"registry": map[string]any{"flush": typed.RegistryFlush}})
		if err != nil {
			return settings{}, err
		}
	}

	return settings{
		Filebeat:           fb,
		EventLogs:          eventLogs,
		RegistryFile:       typed.RegistryFile,
		OverwritePipelines: typed.OverwritePipelines,
	}, nil
}
