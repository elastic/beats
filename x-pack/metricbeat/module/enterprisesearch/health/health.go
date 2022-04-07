// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package health

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v8/metricbeat/helper"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/mb/parse"
)

const (
	// defaultScheme is the default scheme to use when it is not specified in
	// the host config.
	defaultScheme = "http"

	// defaultPath is the default path to the Enterprise Search Health API
	defaultPath = "/api/ent/v1/internal/health"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
	}.Build()
)

func init() {
	mb.Registry.MustAddMetricSet("enterprisesearch", "health", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

type MetricSet struct {
	mb.BaseMetricSet
	http         *helper.HTTP
	XPackEnabled bool
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The Enterprise Search health metricset is currently in beta.")

	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}
	config := struct {
		XPackEnabled bool `config:"xpack.enabled"`
	}{
		XPackEnabled: false,
	}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		base,
		http,
		config.XPackEnabled,
	}, nil
}

// Makes a GET request to Enterprise Search Health API (see defaultPath)
// and generates a monitoring event based on the fetched metrics.
// Returns nil or an error object.
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	content, err := m.http.FetchContent()
	if err != nil {
		return errors.Wrap(err, "error in fetch")
	}

	err = eventMapping(report, content, m.XPackEnabled)
	if err != nil {
		return errors.Wrap(err, "error converting event")
	}

	return nil
}
