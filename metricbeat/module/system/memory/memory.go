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

//go:build darwin || freebsd || linux || openbsd || windows || aix

package memory

import (
	"fmt"
	"runtime"

	"github.com/elastic/beats/v7/libbeat/common/diagnostics"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	metrics "github.com/elastic/beats/v7/pkg/systemmetrics/metric/memory"
	"github.com/elastic/beats/v7/pkg/systemmetrics/metric/system/resolve"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transform/typeconv"
)

func init() {
	mb.Registry.MustAddMetricSet("system", "memory", New,
		mb.WithHostParser(parse.EmptyHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching system memory metrics.
type MetricSet struct {
	mb.BaseMetricSet
	mod resolve.Resolver
}

// New is a mb.MetricSetFactory that returns a memory.MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	sys, ok := base.Module().(resolve.Resolver)
	if !ok {
		return nil, fmt.Errorf("module %T does not implement resolve.Resolver", base.Module())
	}
	return &MetricSet{BaseMetricSet: base, mod: sys}, nil
}

// Fetch fetches memory metrics from the OS.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {

	eventRaw, err := metrics.Get(m.mod)
	if err != nil {
		return fmt.Errorf("error fetching memory metrics: %w", err)
	}

	memory := mapstr.M{}
	err = typeconv.Convert(&memory, &eventRaw)
	if err != nil {
		return err
	}
	r.Event(mb.Event{
		MetricSetFields: memory,
	})

	return nil
}

// Diagnostics implmements the DiagnosticSet interface
func (m *MetricSet) Diagnostics() []diagnostics.DiagnosticSetup {
	m.Logger().Infof("got DiagnosticSetup request for system/memory")
	if runtime.GOOS == "linux" {
		return []diagnostics.DiagnosticSetup{{
			Name:        "memory-meminfo",
			Description: "/proc/meminfo file",
			Filename:    "meminfo",
			Callback:    m.getMemDiagnostic,
		}}
	}
	return nil
}

func (m *MetricSet) getMemDiagnostic() []byte {
	sys, ok := m.Module().(resolve.Resolver)
	if !ok {
		return fmt.Appendf(nil, "Error fetching data: module %T does not implement resolve.Resolver", m.Module())
	}
	return diagnostics.GetRawFileOrErrorString(sys, "/proc/meminfo")
}
