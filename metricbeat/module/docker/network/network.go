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

// +build linux darwin windows

package network

import (
	"runtime"

	"github.com/docker/docker/client"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/docker"
)

func init() {
	mb.Registry.MustAddMetricSet("docker", "network", New,
		mb.WithHostParser(docker.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching docker network metrics.
type MetricSet struct {
	mb.BaseMetricSet
	netService   *NetService
	dockerClient *client.Client
	cfg          Config
}

// New creates a new instance of the docker network MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := DefaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	client, err := docker.NewDockerClient(base.HostData().URI, docker.Config{DeDot: config.DeDot, TLS: config.TLS})
	if err != nil {
		return nil, err
	}

	// Network summary requres a linux procfs system under it to read from the cgroups. Disable reporting otherwise.
	if runtime.GOOS != "linux" {
		base.Logger().Debug("Not running on linux, docker network detailed stats disabled.")
		config.NetworkSummary = false
	}

	return &MetricSet{
		BaseMetricSet: base,
		dockerClient:  client,
		netService: &NetService{
			NetworkStatPerContainer: make(map[string]map[string]NetRaw),
		},
		cfg: config,
	}, nil
}

// Fetch methods creates a list of network events for each container.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	stats, err := docker.FetchStats(m.dockerClient, m.Module().Config().Timeout)
	if err != nil {
		return errors.Wrap(err, "failed to get docker stats")
	}

	formattedStats, err := m.netService.getNetworkStatsPerContainer(m.dockerClient, m.Module().Config().Timeout, stats, m.cfg)
	if err != nil {
		return errors.Wrap(err, "error fetching container network stats")
	}
	eventsMapping(r, formattedStats, m.cfg.NetworkSummary)

	return nil
}

//Close stops the metricset
func (m *MetricSet) Close() error {

	return m.dockerClient.Close()
}
