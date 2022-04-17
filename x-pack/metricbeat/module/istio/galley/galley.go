// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package galley

import (
	"github.com/menderesk/beats/v7/metricbeat/helper/prometheus"
	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/mb/parse"
)

const (
	defaultScheme = "http"
	defaultPath   = "/metrics"
)

var (
	hostParser = parse.URLHostParserBuilder{
		DefaultScheme: defaultScheme,
		DefaultPath:   defaultPath,
	}.Build()
)

var mapping = &prometheus.MetricsMapping{
	Metrics: map[string]prometheus.MetricMap{
		"galley_istio_authentication_meshpolicies":                         prometheus.Metric("istio.authentication.meshpolicies"),
		"galley_istio_authentication_policies":                             prometheus.Metric("istio.authentication.policies"),
		"galley_istio_mesh_MeshConfig":                                     prometheus.Metric("istio.mesh.MeshConfig"),
		"galley_istio_networking_destinationrules":                         prometheus.Metric("istio.networking.destinationrules"),
		"galley_istio_networking_envoyfilters":                             prometheus.Metric("istio.networking.envoyfilters"),
		"galley_istio_networking_gateways":                                 prometheus.Metric("istio.networking.gateways"),
		"galley_istio_networking_sidecars":                                 prometheus.Metric("istio.networking.sidecars"),
		"galley_istio_networking_virtualservices":                          prometheus.Metric("istio.networking.virtualservices"),
		"galley_istio_policy_attributemanifests":                           prometheus.Metric("istio.policy.attributemanifests"),
		"galley_istio_policy_handlers":                                     prometheus.Metric("istio.policy.handlers"),
		"galley_istio_policy_instances":                                    prometheus.Metric("istio.policy.instances"),
		"galley_istio_policy_rules":                                        prometheus.Metric("istio.policy.rules"),
		"galley_runtime_processor_event_span_duration_milliseconds":        prometheus.Metric("runtime.processor.event_span.duration.ms"),
		"galley_runtime_processor_snapshot_events_total":                   prometheus.Metric("runtime.processor.snapshot_events"),
		"galley_runtime_processor_snapshot_lifetime_duration_milliseconds": prometheus.Metric("runtime.processor.snapshot_lifetime.duration.ms"),
		"galley_runtime_state_type_instances_total":                        prometheus.Metric("runtime.state_type_instances"),
		"galley_runtime_strategy_on_change_total":                          prometheus.Metric("runtime.strategy.on_change"),
		"galley_runtime_strategy_timer_quiesce_reached_total":              prometheus.Metric("runtime.strategy.timer_quiesce_reached"),
		"galley_source_kube_event_success_total":                           prometheus.Metric("source_kube_event_success_total"),
		"galley_validation_cert_key_updates":                               prometheus.Metric("validation.cert_key.updates"),
		"galley_validation_config_load":                                    prometheus.Metric("validation.config.load"),
		"galley_validation_config_updates":                                 prometheus.Metric("validation.config.updates"),
	},

	Labels: map[string]prometheus.LabelMap{
		"name":       prometheus.KeyLabel("name"),
		"namespace":  prometheus.KeyLabel("namespace"),
		"version":    prometheus.KeyLabel("version"),
		"collection": prometheus.KeyLabel("collection"),
	},
}

func init() {
	mb.Registry.MustAddMetricSet("istio", "galley",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(hostParser))
}
