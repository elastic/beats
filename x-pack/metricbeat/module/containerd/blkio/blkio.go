// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package blkio

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/metricbeat/module/containerd"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"

	"github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	defaultPath   = "/v1/metrics"
)

var (
	// HostParser validates Prometheus URLs
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
		PathConfigKey: "metrics_path",
	}.Build()

	// Mapping of state metrics
	mapping = &prometheus.MetricsMapping{
		Metrics: map[string]prometheus.MetricMap{
			"container_blkio_io_serviced_recursive_total": prometheus.Metric("", prometheus.OpFilterMap(
				"op", map[string]string{
					"Read":  "read.ops",
					"Write": "write.ops",
					"Total": "summary.ops",
				},
			)),
			"container_blkio_io_service_bytes_recursive_bytes": prometheus.Metric("", prometheus.OpFilterMap(
				"op", map[string]string{
					"Read":  "read.bytes",
					"Write": "write.bytes",
					"Total": "summary.bytes",
				},
			)),
		},
		Labels: map[string]prometheus.LabelMap{
			"container_id": prometheus.KeyLabel("id"),
			"device":       prometheus.KeyLabel("device"),
			"namespace":    prometheus.KeyLabel("namespace"),
		},
	}
)

// Metricset for containerd blkio is a prometheus based metricset
type metricset struct {
	mb.BaseMetricSet
	prometheusClient prometheus.Prometheus
	mod              containerd.Module
}

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	mb.Registry.MustAddMetricSet("containerd", "blkio", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The containerd blkio metricset is beta.")

	pc, err := prometheus.NewPrometheusClient(base)
	if err != nil {
		return nil, err
	}

	mod, ok := base.Module().(containerd.Module)
	if !ok {
		return nil, fmt.Errorf("must be child of kubernetes module")
	}
	return &metricset{
		BaseMetricSet:    base,
		prometheusClient: pc,
		mod:              mod,
	}, nil
}

// Fetch gathers information from the containerd and reports events with this information.
func (m *metricset) Fetch(reporter mb.ReporterV2) error {
	families, _, err := m.mod.GetContainerdMetricsFamilies(m.prometheusClient)
	if err != nil {
		return fmt.Errorf("error getting families: %w", err)
	}
	events, err := m.prometheusClient.ProcessMetrics(families, mapping)
	if err != nil {
		return fmt.Errorf("error getting events: %w", err)
	}
	for _, event := range events {
		// setting ECS container.id and module field containerd.namespace
		containerFields := mapstr.M{}
		moduleFields := mapstr.M{}
		rootFields := mapstr.M{}

		cID := containerd.GetAndDeleteCid(event)
		namespace := containerd.GetAndDeleteNamespace(event)

		// Add container.id ECS field
		_, _ = containerFields.Put("id", cID)
		// Add disk.read.bytes ECS field
		diskReadBytes, err := event.GetValue("read.bytes")
		if err == nil {
			_, _ = containerFields.Put("disk.read.bytes", diskReadBytes)
		}
		// Add disk.write.bytes ECS field
		diskWriteBytes, err := event.GetValue("write.bytes")
		if err == nil {
			_, _ = containerFields.Put("disk.write.bytes", diskWriteBytes)
		}

		_, _ = rootFields.Put("container", containerFields)
		_, _ = moduleFields.Put("namespace", namespace)

		reporter.Event(mb.Event{
			RootFields:      rootFields,
			ModuleFields:    moduleFields,
			MetricSetFields: event,
			Namespace:       "containerd.blkio",
		})
	}
	return nil
}
