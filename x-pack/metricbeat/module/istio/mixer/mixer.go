// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mixer

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
		"mixer_config_adapter_info_config_errors_total":   prometheus.Metric("config.adapter.info.errors.config"),
		"mixer_config_adapter_info_configs_total":         prometheus.Metric("config.adapter.info.configs"),
		"mixer_config_attributes_total":                   prometheus.Metric("config.attributes"),
		"mixer_config_handler_configs_total":              prometheus.Metric("config.handler.configs"),
		"mixer_config_handler_validation_error_total":     prometheus.Metric("config.handler.errors.validation"),
		"mixer_config_instance_config_errors_total":       prometheus.Metric("config.instance.errors.config"),
		"mixer_config_instance_configs_total":             prometheus.Metric("config.instance.configs"),
		"mixer_config_rule_config_errors_total":           prometheus.Metric("config.rule.errors.config"),
		"mixer_config_rule_config_match_error_total":      prometheus.Metric("config.rule.errors.match"),
		"mixer_config_rule_configs_total":                 prometheus.Metric("config.rule.configs"),
		"mixer_config_template_config_errors_total":       prometheus.Metric("config.template.errors.config"),
		"mixer_config_template_configs_total":             prometheus.Metric("config.template.configs"),
		"mixer_config_unsatisfied_action_handler_total":   prometheus.Metric("config.unsatisfied.action_handler"),
		"mixer_dispatcher_destinations_per_variety_total": prometheus.Metric("dispatcher_destinations_per_variety_total"),
		"mixer_handler_closed_handlers_total":             prometheus.Metric("handler.handlers.closed"),
		"mixer_handler_daemons_total":                     prometheus.Metric("handler.daemons"),
		"mixer_handler_handler_build_failures_total":      prometheus.Metric("handler.failures.build"),
		"mixer_handler_handler_close_failures_total":      prometheus.Metric("handler.failures.close"),
		"mixer_handler_new_handlers_total":                prometheus.Metric("handler.handlers.new"),
		"mixer_handler_reused_handlers_total":             prometheus.Metric("handler.handlers.reused"),
		"istio_mcp_request_acks_total":                    prometheus.Metric("istio.mcp.request.acks"),
	},

	Labels: map[string]prometheus.LabelMap{
		"handler": prometheus.KeyLabel("handler.name"),
		"variety": prometheus.KeyLabel("variety"),
	},
}

func init() {
	mb.Registry.MustAddMetricSet("istio", "mixer",
		prometheus.MetricSetBuilder(mapping),
		mb.WithHostParser(hostParser))
}
