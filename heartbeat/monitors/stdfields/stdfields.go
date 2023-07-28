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

package stdfields

import (
	"fmt"
	"time"

	hbconfig "github.com/elastic/beats/v7/heartbeat/config"
	"github.com/elastic/beats/v7/heartbeat/scheduler/schedule"
	"github.com/elastic/elastic-agent-libs/config"
)

type ServiceFields struct {
	Name string `config:"name"`
}

// StdMonitorFields represents the generic configuration options around a monitor plugin.
type StdMonitorFields struct {
	ID                string             `config:"id"`
	Name              string             `config:"name"`
	Type              string             `config:"type" validate:"required"`
	Schedule          *schedule.Schedule `config:"schedule" validate:"required"`
	Timeout           time.Duration      `config:"timeout"`
	Service           ServiceFields      `config:"service"`
	Origin            string             `config:"origin"`
	LegacyServiceName string             `config:"service_name"`
	MaxAttempts       uint16             `config:"max_attempts"`
	// Used by zip_url and local monitors
	// kibana originating monitors only run one journey at a time
	// and just use the `fields` syntax / manually set monitor IDs
	IsLegacyBrowserSource bool
	Enabled               bool `config:"enabled"`
	// TODO: Delete this once browser / local monitors are removed
	Source struct {
		ZipUrl *config.C `config:"zip_url"`
		Local  *config.C `config:"local"`
	} `config:"source"`
	RunFrom *hbconfig.LocationWithID `config:"run_from"`
	// Set to true by monitor.go if monitor configuration is unrunnable
	// Maybe there's a more elegant way to handle this
	BadConfig bool
}

func ConfigToStdMonitorFields(conf *config.C) (StdMonitorFields, error) {
	sFields := StdMonitorFields{Enabled: true, MaxAttempts: 1}

	if err := conf.Unpack(&sFields); err != nil {
		return sFields, fmt.Errorf("error unpacking monitor plugin config: %w", err)
	}

	// Use `service_name` if `service.name` is unspecified
	// `service_name` was only document in the 7.10.0 release.
	if sFields.LegacyServiceName != "" {
		if sFields.Service.Name == "" {
			sFields.Service.Name = sFields.LegacyServiceName
		}
	}

	// TODO: Delete this once browser / local monitors are removed
	if sFields.Source.Local != nil || sFields.Source.ZipUrl != nil {
		sFields.IsLegacyBrowserSource = true
	}

	return sFields, nil
}
