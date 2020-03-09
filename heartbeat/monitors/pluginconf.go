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
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/heartbeat/scheduler/schedule"
	"github.com/elastic/beats/v7/libbeat/common"
)

// ErrPluginDisabled is returned when the monitor plugin is marked as disabled.
var ErrPluginDisabled = errors.New("Monitor not loaded, plugin is disabled")

// MonitorPluginInfo represents the generic configuration options around a monitor plugin.
type MonitorPluginInfo struct {
	ID       string             `config:"id"`
	Name     string             `config:"name"`
	Type     string             `config:"type" validate:"required"`
	Schedule *schedule.Schedule `config:"schedule" validate:"required"`
	Timeout  time.Duration      `config:"timeout"`
	Enabled  bool               `config:"enabled"`
}

func pluginInfo(config *common.Config) (MonitorPluginInfo, error) {
	mpi := MonitorPluginInfo{Enabled: true}

	if err := config.Unpack(&mpi); err != nil {
		return mpi, errors.Wrap(err, "error unpacking monitor plugin config")
	}

	if !mpi.Enabled {
		return mpi, ErrPluginDisabled
	}

	return mpi, nil
}
