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

package diskio

import (
	"fmt"

	"github.com/docker/docker/client"

	"github.com/elastic/beats/v8/libbeat/logp"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/module/docker"
)

func init() {
	mb.Registry.MustAddMetricSet("docker", "diskio", New,
		mb.WithHostParser(docker.HostParser),
		mb.DefaultMetricSet(),
	)
}

// Config "imports" the base module-level config, plus our metricset options
type Config struct {
	TLS       *docker.TLSConfig `config:"ssl"`
	DeDot     bool              `config:"labels.dedot"`
	SkipMajor []uint64          `config:"skip_major"`
}

// The major devices we'll skip by default. 9 == mdraid, 253 == device-mapper
var defaultMajorDev = []uint64{9, 253}

func defaultConfig() Config {
	//This is a bit awkward, but the config Unwrap() function is a bit awkward in that it will only partly overwrite
	// an array value, which makes handling the `skip_major` array a bit annoying.
	parentDefault := docker.DefaultConfig()
	return Config{
		TLS:   parentDefault.TLS,
		DeDot: parentDefault.DeDot,
	}
}

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	blkioService *BlkioService
	dockerClient *client.Client
	config       Config
}

// New create a new instance of the docker diskio MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig()
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	if config.SkipMajor == nil {
		config.SkipMajor = defaultMajorDev
	}
	logp.L().Debugf("Skipping major devices: %v", config.SkipMajor)
	client, err := docker.NewDockerClient(base.HostData().URI, docker.Config{TLS: config.TLS, DeDot: config.DeDot})
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		dockerClient:  client,
		blkioService:  NewBlkioService(),
		config:        config,
	}, nil
}

// Fetch creates list of events with diskio stats for all containers.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	stats, err := docker.FetchStats(m.dockerClient, m.Module().Config().Timeout)
	if err != nil {
		return fmt.Errorf("failed to get docker stats: %w", err)
	}

	formattedStats := m.blkioService.getBlkioStatsList(stats, m.config.DeDot, m.config.SkipMajor)
	eventsMapping(r, formattedStats)

	return nil
}

//Close stops the metricset
func (m *MetricSet) Close() error {

	return m.dockerClient.Close()
}
