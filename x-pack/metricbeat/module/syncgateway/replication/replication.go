// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package replication

import (
	"fmt"

	"github.com/elastic/beats/v8/metricbeat/helper"
	"github.com/elastic/beats/v8/metricbeat/mb"
	"github.com/elastic/beats/v8/metricbeat/mb/parse"
	"github.com/elastic/beats/v8/x-pack/metricbeat/module/syncgateway"
)

const (
	defaultScheme = "http"
	defaultPath   = "/_expvar"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
	}.Build()
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("syncgateway", "replication", New,
		mb.WithHostParser(hostParser),
	)
}

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	http *helper.HTTP
	mod  syncgateway.Module
}

// New create a new instance of the MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	mod, ok := base.Module().(syncgateway.Module)
	if !ok {
		return nil, fmt.Errorf("error trying to type assert syncgateway.Module. Base module must be composed of syncgateway.Module")
	}

	http, err := helper.NewHTTP(base)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		http:          http,
		mod:           mod,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(reporter mb.ReporterV2) error {
	response, err := m.mod.GetSyncgatewayResponse(m.http)
	if err != nil {
		return err
	}

	eventMapping(reporter, response)

	return nil
}
