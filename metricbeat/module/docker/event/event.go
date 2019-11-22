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

package event

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/docker"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("docker", "event", New,
		mb.WithHostParser(docker.HostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	dockerClient *client.Client
	dedot        bool
	logger       *logp.Logger
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
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
		logger:        logp.NewLogger("docker"),
	}, nil
}

// Run listens for docker events and reports them
func (m *MetricSet) Run(ctx context.Context, reporter mb.ReporterV2) {
	options := types.EventsOptions{
		Since: fmt.Sprintf("%d", time.Now().Unix()),
	}

	defer m.dockerClient.Close()

	for {
		events, errors := m.dockerClient.Events(ctx, options)

	WATCH:
		for {
			select {
			case event := <-events:
				m.logger.Debug("Got a new docker event: %v", event)
				m.reportEvent(reporter, event)

			case err := <-errors:
				// An error can be received on context cancellation, don't reconnect
				// if context is done.
				select {
				case <-ctx.Done():
					m.logger.Debug("docker", "Event watcher stopped")
					return
				default:
				}
				// Restart watch call
				m.logger.Errorf("Error watching for docker events: %v", err)
				time.Sleep(1 * time.Second)
				break WATCH

			case <-ctx.Done():
				m.logger.Debug("docker", "Event watcher stopped")
				return
			}
		}
	}
}

func (m *MetricSet) reportEvent(reporter mb.ReporterV2, event events.Message) {
	time := time.Unix(event.Time, 0)

	attributes := make(map[string]string, len(event.Actor.Attributes))
	for k, v := range event.Actor.Attributes {
		if m.dedot {
			k = common.DeDot(k)
		}
		attributes[k] = v
	}

	reporter.Event(mb.Event{
		Timestamp: time,
		MetricSetFields: common.MapStr{
			"id":     event.ID,
			"type":   event.Type,
			"action": event.Action,
			"status": event.Status,
			"from":   event.From,
			"actor": common.MapStr{
				"id":         event.Actor.ID,
				"attributes": attributes,
			},
		},
	})
}
