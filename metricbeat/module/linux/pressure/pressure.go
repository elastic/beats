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

package pressure

import (
	"fmt"
	"runtime"

	"github.com/pkg/errors"
	"github.com/prometheus/procfs"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

const (
	moduleName    = "linux"
	metricsetName = "pressure"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet(moduleName, metricsetName, New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	mod resolve.Resolver
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta(fmt.Sprintf("The %s %s metricset is beta.", moduleName, metricsetName))

	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("the %v/%v metricset is only supported on Linux", moduleName, metricsetName)
	}

	sys := base.Module().(resolve.Resolver)

	return &MetricSet{
		BaseMetricSet: base,
		mod:           sys,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	events, err := fetchLinuxPSIStats(m)
	if err != nil {
		return errors.Wrap(err, "error fetching PSI stats")
	}

	for _, event := range events {
		report.Event(mb.Event{
			MetricSetFields: event,
		})
	}
	return nil
}

func fetchLinuxPSIStats(m *MetricSet) ([]common.MapStr, error) {
	resources := []string{"cpu", "memory", "io"}
	events := []common.MapStr{}

	procfs, err := procfs.NewFS(m.mod.ResolveHostFS("/proc"))
	if err != nil {
		return nil, errors.Wrapf(err, "error creating new Host FS at %s", m.mod.ResolveHostFS("/proc"))
	}

	for _, resource := range resources {
		psiMetric, err := procfs.PSIStatsForResource(resource)
		if err != nil {
			return nil, errors.Wrap(err, "check that /proc/pressure is available, and/or enabled")
		}

		event := common.MapStr{
			resource: common.MapStr{
				"some": common.MapStr{
					"10": common.MapStr{
						"pct": psiMetric.Some.Avg10,
					},
					"60": common.MapStr{
						"pct": psiMetric.Some.Avg60,
					},
					"300": common.MapStr{
						"pct": psiMetric.Some.Avg300,
					},
					"total": common.MapStr{
						"time": common.MapStr{
							"us": psiMetric.Some.Total,
						},
					},
				},
			},
		}

		// /proc/pressure/cpu does not contain 'full' metrics
		if resource != "cpu" {
			event.Put(resource+".full.10.pct", psiMetric.Full.Avg10)
			event.Put(resource+".full.60.pct", psiMetric.Full.Avg60)
			event.Put(resource+".full.300.pct", psiMetric.Full.Avg300)
			event.Put(resource+".full.total.time.us", psiMetric.Full.Total)
		}

		events = append(events, event)
	}
	return events, nil
}
