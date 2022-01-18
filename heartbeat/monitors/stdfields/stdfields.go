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

	"github.com/elastic/beats/v7/heartbeat/scheduler/schedule"
	"github.com/elastic/beats/v7/libbeat/common"
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
	LegacyServiceName string             `config:"service_name"`
	Enabled           bool               `config:"enabled"`
}

func ConfigToStdMonitorFields(config *common.Config) (StdMonitorFields, error) {
	mpi := StdMonitorFields{Enabled: true}

	if err := config.Unpack(&mpi); err != nil {
		return mpi, fmt.Errorf("error unpacking monitor plugin config: %w", err)
	}

	// Use `service_name` if `service.name` is unspecified
	// `service_name` was only document in the 7.10.0 release.
	if mpi.LegacyServiceName != "" {
		if mpi.Service.Name == "" {
			mpi.Service.Name = mpi.LegacyServiceName
		}
	}

	return mpi, nil
}
