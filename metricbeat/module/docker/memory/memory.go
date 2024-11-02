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

//go:build linux || darwin || windows

package memory

import (
	"fmt"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/docker/docker/client"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/docker"
)

func init() {
	mb.Registry.MustAddMetricSet("docker", "memory", New,
		mb.WithHostParser(docker.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	memoryService *MemoryService
	dockerClient  *client.Client
	dedot         bool
	logger        *logp.Logger
}

// New creates a new instance of the docker memory MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	logger := logp.NewLogger("docker.memory")
	config := docker.DefaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	dockerClient, err := docker.NewDockerClient(base.HostData().URI, config)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		memoryService: &MemoryService{},
		dockerClient:  dockerClient,
		dedot:         config.DeDot,
		logger:        logger,
	}, nil
}

// Fetch creates a list of memory events for each container.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	stats, err := docker.FetchStats(m.dockerClient, m.Module().Config().Timeout)
	if err != nil {
		return fmt.Errorf("failed to get docker stats: %w", err)
	}

	memoryStats := m.memoryService.getMemoryStatsList(stats, m.dedot)
	if len(memoryStats) == 0 {
		// No memory stats available, probably
		// because no containers are running.
		m.logger.Debug("No memory stats data available")
		return nil
	}
	eventsMapping(r, memoryStats)

	return nil
}

// Close stops the metricset
func (m *MetricSet) Close() error {

	return m.dockerClient.Close()
}
