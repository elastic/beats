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

// Package config provides the winlogbeat specific configuration options.
package config

import (
	"fmt"
	"time"

	"github.com/joeshaw/multierror"

	"github.com/menderesk/beats/v7/libbeat/common"
)

const (
	// DefaultRegistryFile specifies the default filename of the registry file.
	DefaultRegistryFile = ".winlogbeat.yml"
)

var DefaultSettings = WinlogbeatConfig{
	RegistryFile:  DefaultRegistryFile,
	RegistryFlush: 5 * time.Second,
}

// WinlogbeatConfig contains all of Winlogbeat configuration data.
type WinlogbeatConfig struct {
	EventLogs          []*common.Config `config:"event_logs"`
	RegistryFile       string           `config:"registry_file"`
	RegistryFlush      time.Duration    `config:"registry_flush"`
	ShutdownTimeout    time.Duration    `config:"shutdown_timeout"`
	OverwritePipelines bool             `config:"overwrite_pipelines"`
}

// Validate validates the WinlogbeatConfig data and returns an error describing
// all problems or nil if there are none.
func (ebc WinlogbeatConfig) Validate() error {
	var errs multierror.Errors

	if len(ebc.EventLogs) == 0 {
		errs = append(errs, fmt.Errorf("at least one event log must be "+
			"configured as part of event_logs"))
	}

	return errs.Err()
}
