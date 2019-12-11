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

// +build darwin freebsd linux windows

package network

import (
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"

	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/net"
)

var debugf = logp.MakeDebug("system-network")

func init() {
	mb.Registry.MustAddMetricSet("system", "network", New,
		mb.WithHostParser(parse.EmptyHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching system network IO metrics.
type MetricSet struct {
	mb.BaseMetricSet
	interfaces map[string]struct{}
}

// New is a mb.MetricSetFactory that returns a new MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	// Unpack additional configuration options.
	config := struct {
		Interfaces []string `config:"interfaces"`
	}{}
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	var interfaceSet map[string]struct{}
	if len(config.Interfaces) > 0 {
		interfaceSet = make(map[string]struct{}, len(config.Interfaces))
		for _, ifc := range config.Interfaces {
			interfaceSet[strings.ToLower(ifc)] = struct{}{}
		}
		debugf("network io stats will be included for %v", interfaceSet)
	}

	return &MetricSet{
		BaseMetricSet: base,
		interfaces:    interfaceSet,
	}, nil
}

// Fetch fetches network IO metrics from the OS.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	stats, err := net.IOCounters(true)
	if err != nil {
		return errors.Wrap(err, "network io counters")
	}

	for _, counters := range stats {
		if m.interfaces != nil {
			// Select stats by interface name.
			name := strings.ToLower(counters.Name)
			if _, include := m.interfaces[name]; !include {
				continue
			}
		}

		isOpen := r.Event(mb.Event{
			MetricSetFields: ioCountersToMapStr(counters),
		})
		if !isOpen {
			return nil
		}
	}

	return nil
}

func ioCountersToMapStr(counters net.IOCountersStat) common.MapStr {
	return common.MapStr{
		"name": counters.Name,
		"in": common.MapStr{
			"errors":  counters.Errin,
			"dropped": counters.Dropin,
			"bytes":   counters.BytesRecv,
			"packets": counters.PacketsRecv,
		},
		"out": common.MapStr{
			"errors":  counters.Errout,
			"dropped": counters.Dropout,
			"packets": counters.PacketsSent,
			"bytes":   counters.BytesSent,
		},
	}
}
