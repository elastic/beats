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

//go:build darwin || freebsd || linux || openbsd || windows
// +build darwin freebsd linux openbsd windows

package filesystem

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
	"github.com/elastic/elastic-agent-libs/mapstr"
	fs "github.com/elastic/elastic-agent-system-metrics/metric/system/filesystem"
)

func init() {
	mb.Registry.MustAddMetricSet("system", "filesystem", New,
		mb.WithHostParser(parse.EmptyHostParser),
	)
}

// Config stores the metricset-local config
type Config struct {
	IgnoreTypes []string `config:"filesystem.ignore_types"`
}

// MetricSet for fetching filesystem metrics.
type MetricSet struct {
	mb.BaseMetricSet
	config Config
	sys    resolve.Resolver
}

// New creates and returns a new instance of MetricSet.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	var config Config
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	sys, ok := base.Module().(resolve.Resolver)
	if !ok {
		return nil, fmt.Errorf("resolver cannot be cast from the module")
	}

	if config.IgnoreTypes == nil {
		config.IgnoreTypes = fs.DefaultIgnoredTypes(sys)
	}
	if len(config.IgnoreTypes) > 0 {
		logp.Info("Ignoring filesystem types: %s", strings.Join(config.IgnoreTypes, ", "))
	}
	return &MetricSet{
		BaseMetricSet: base,
		config:        config,
		sys:           sys,
	}, nil
}

// Fetch fetches filesystem metrics for all mounted filesystems and returns
// an event for each mount point.
func (m *MetricSet) Fetch(r mb.ReporterV2) error {

	fsList, err := fs.GetFilesystems(m.sys, fs.BuildFilterWithList(m.config.IgnoreTypes))
	if err != nil {
		return fmt.Errorf("error fetching filesystem list: %w", err)
	}

	for _, fs := range fsList {
		err := fs.GetUsage()
		if err != nil {
			return fmt.Errorf("error getting filesystem usage for %s: %w", fs.Directory, err)
		}
		out := mapstr.M{}
		err = typeconv.Convert(&out, fs)
		if err != nil {
			return fmt.Errorf("error converting event %s: %w", fs.Device, err)
		}

		event := mb.Event{
			MetricSetFields: out,
		}
		if !r.Event(event) {
			return nil
		}
	}
	return nil
}
