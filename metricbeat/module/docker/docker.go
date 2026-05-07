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

package docker

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/docker/go-connections/tlsconfig"
	"github.com/moby/moby/api/types/container"
	dockerclient "github.com/moby/moby/client"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/elastic-agent-autodiscover/docker"
	"github.com/elastic/elastic-agent-libs/logp"
)

// HostParser is a TCP host parser function for docker tcp host addresses
var HostParser = parse.URLHostParserBuilder{DefaultScheme: "tcp"}.Build()

func init() {
	// Register the ModuleFactory function for the "docker" module.
	if err := mb.Registry.AddModule("docker", NewModule); err != nil {
		panic(err)
	}
}

// NewModule creates a new module after performing validation.
func NewModule(base mb.BaseModule) (mb.Module, error) {
	// Validate that at least one host has been specified.
	config := struct {
		Hosts []string `config:"hosts"    validate:"nonzero,required"`
	}{}
	if err := base.UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &base, nil
}

// NewDockerClient initializes and returns a new Docker client
func NewDockerClient(endpoint string, config Config, logger *logp.Logger) (*dockerclient.Client, error) {
	var httpClient *http.Client

	if config.TLS.IsEnabled() {
		options := tlsconfig.Options{
			CAFile:   config.TLS.CA,
			CertFile: config.TLS.Certificate,
			KeyFile:  config.TLS.Key,
		}

		tlsc, err := tlsconfig.Client(options)
		if err != nil {
			return nil, err
		}

		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsc,
			},
		}
	}

	client, err := docker.NewClient(endpoint, httpClient, nil, logger)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// FetchStats returns a list of running containers with all related stats inside
func FetchStats(cli *dockerclient.Client, timeout time.Duration, stream bool, logger *logp.Logger) ([]Stat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	listResult, err := cli.ContainerList(ctx, dockerclient.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup

	containersList := make([]Stat, 0, len(listResult.Items))
	statsQueue := make(chan Stat, 1)
	wg.Add(len(listResult.Items))

	for _, c := range listResult.Items {
		go func(c container.Summary) {
			defer wg.Done()
			statsQueue <- exportContainerStats(ctx, cli, &c, stream, logger)
		}(c)
	}

	go func() {
		wg.Wait()
		close(statsQueue)
	}()

	// This will break after the queue has been drained and queue is closed.
	for stat := range statsQueue {
		// If names is empty, there is not data inside
		if len(stat.Container.Names) != 0 {
			containersList = append(containersList, stat)
		}
	}

	return containersList, err
}

// exportContainerStats loads stats for the given container
//
// This is currently very inefficient as docker calculates the average for each request,
// means each request will take at least 2s: https://github.com/docker/docker/blob/master/cli/command/container/stats_helpers.go#L148
// Getting all stats at once is implemented here: https://github.com/docker/docker/pull/25361
// In case stream is true, we use get a stream of results for container stats. From the stream we keep the second result.
// This is needed for podman use case where in case stream is false, no precpu stats are returned. The precpu stats
// are required for the cpu percentage calculation. We keep the second  result as in the first result, the stats are not correct.
func exportContainerStats(ctx context.Context, cli *dockerclient.Client, cont *container.Summary, stream bool, logger *logp.Logger) Stat {
	var event Stat
	event.Container = cont
	containerStats, err := cli.ContainerStats(ctx, cont.ID, dockerclient.ContainerStatsOptions{Stream: stream, IncludePreviousSample: !stream})
	if err != nil {
		logger.Debugf("Failed fetching container stats: %v", err)
		return event
	}
	defer containerStats.Body.Close()

	// JSON decoder
	decoder := json.NewDecoder(containerStats.Body)
	if !stream {
		if err := decoder.Decode(&event.Stats); err != nil {
			logger.Debugf("Failed decoding event: %v", err)
			return event
		}
	} else {
		// handle stream. Take the second result.
		count := 0
		for decoder.More() {
			if err := decoder.Decode(&event.Stats); err != nil {
				logger.Debugf("Failed decoding event: %v", err)
				return event
			}

			count++
			// Exit after the second result
			if count == 2 {
				break
			}
		}
	}
	return event
}
