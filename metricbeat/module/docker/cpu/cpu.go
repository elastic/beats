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
// +build linux darwin windows

package cpu

import (
	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/docker"
)

func init() {
	mb.Registry.MustAddMetricSet("docker", "cpu", New,
		mb.WithHostParser(docker.HostParser),
		mb.DefaultMetricSet(),
	)
}

type MetricSet struct {
	mb.BaseMetricSet
	cpuService   *CPUService
	dockerClient *client.Client
	dedot        bool
}

// New creates a new instance of the docker cpu MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := docker.DefaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	client, err := docker.NewDockerClient(base.HostData().URI, config)
	if err != nil {
		return nil, err
	}

	cpuConfig := struct {
		Cores bool `config:"cpu.cores"`
	}{
		Cores: true,
	}
	if err := base.Module().UnpackConfig(&cpuConfig); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		dockerClient:  client,
		cpuService:    &CPUService{Cores: cpuConfig.Cores},
		dedot:         config.DeDot,
	}, nil
}

// Fetch returns a list of docker CPU stats.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	stats, err := docker.FetchStats(m.dockerClient, m.Module().Config().Timeout)
	if err != nil {
		return errors.Wrap(err, "failed to get docker stats")
	}

	formattedStats := m.cpuService.getCPUStatsList(stats, m.dedot)
	eventsMapping(r, formattedStats)

	return nil
}

//Close stops the metricset
func (m *MetricSet) Close() error {

	return m.dockerClient.Close()
}
