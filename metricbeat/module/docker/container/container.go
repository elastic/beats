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

package container

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/module/docker"
)

func init() {
	mb.Registry.MustAddMetricSet("docker", "container", New,
		mb.WithHostParser(docker.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	dockerClient *client.Client
	dedot        bool
}

// New creates a new instance of the docker container MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := docker.DefaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	client, err := docker.NewDockerClient(base.HostData().URI, config)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		dockerClient:  client,
		dedot:         config.DeDot,
	}, nil
}

// Fetch returns a list of all containers as events.
// This is based on https://docs.docker.com/engine/reference/api/docker_remote_api_v1.24/#/list-containers.
func (m *MetricSet) Fetch(ctx context.Context, r mb.ReporterV2) error {
	// Fetch a list of all containers.
	containers, err := m.dockerClient.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return fmt.Errorf("failed to get docker containers list: %w", err)
	}
	eventsMapping(r, containers, m.dedot, m.Logger())

	return nil
}

//Close stops the metricset
func (m *MetricSet) Close() error {

	return m.dockerClient.Close()
}
