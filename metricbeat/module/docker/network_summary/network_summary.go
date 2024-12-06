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

//go:build linux

package network_summary

import (
	"context"
	"fmt"
	"runtime"

	"github.com/docker/docker/client"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/module/docker"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/network"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("docker", "network_summary", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	dockerClient *client.Client
	cfg          Config
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The docker network_summary metricset is beta.")

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
		cfg:           config,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(ctx context.Context, report mb.ReporterV2) error {

	stats, err := docker.FetchStats(m.dockerClient, m.Module().Config().Timeout)
	if err != nil {
		return fmt.Errorf("failed to get docker stats: %w", err)
	}

	for _, myStats := range stats {

		ctx, cancel := context.WithTimeout(ctx, m.Module().Config().Timeout)
		defer cancel()

		inspect, err := m.dockerClient.ContainerInspect(ctx, myStats.Container.ID)
		if err != nil {
			return fmt.Errorf("error fetching stats for container %s: %w", myStats.Container.ID, err)
		}

		rootPID := inspect.ContainerJSONBase.State.Pid

		netNS, err := fetchNamespace(rootPID)
		if err != nil {
			return fmt.Errorf("error fetching namespace for PID %d: %w", rootPID, err)
		}

		networkStats, err := fetchContainerNetStats(m.dockerClient, m.Module().Config().Timeout, myStats.Container.ID)
		if err != nil {
			return fmt.Errorf("error fetching per-PID stats")
		}

		summary := network.MapProcNetCounters(networkStats)
		// attach metadata associated with the network counters
		summary["namespace"] = mapstr.M{
			"id":  netNS,
			"pid": rootPID,
		}

		report.Event(mb.Event{
			RootFields:      docker.NewContainer(myStats.Container, m.cfg.DeDot).ToMapStr(),
			MetricSetFields: summary,
		})

	}

	return nil
}
