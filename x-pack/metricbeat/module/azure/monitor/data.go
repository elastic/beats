// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitor

import (
	"fmt"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"strings"
)

// eventsMapping will map metric values to beats events
func eventsMapping(report mb.ReporterV2, metrics []Metric) error {
	for _, metric := range metrics {
		if len(metric.values) == 0 {
			continue
		}
		metricList := common.MapStr{}

		for _, value := range metric.values {
			metricNameString := fmt.Sprintf("%s", manageMetricName(value.name))
			if value.min != nil {
				metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "min"), *value.min)
			}
			if value.max != nil {
				metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "max"), *value.max)
			}
			if value.average != nil {
				metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "avg"), value.average)
			}
			if value.total != nil {
				metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "total"), value.total)
			}
			if value.count != nil {
				metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "count"), value.count)
			}
		}
		event := mb.Event{
			MetricSetFields: common.MapStr{
				"resource": common.MapStr{
					"name":     metric.resource.Name,
					"location": metric.resource.Location,
					"type":     metric.resource.Type,
				},
				"namespace":      metric.namespace,
				"subscriptionId": "unique identifier",
				"metrics":        metricList.Flatten(),
			},
		}
		if len(metric.dimensions) > 0 {
			for _, dimension := range metric.dimensions {
				event.MetricSetFields.Put(fmt.Sprintf("dimensions.%s", dimension.name), dimension.value)
			}
		}
		report.Event(event)
	}
	return nil
}

func manageMetricName(metric string) string {
	resultMetricName := strings.Replace(metric, " ", "_", -1)
	resultMetricName = strings.Replace(resultMetricName, "/", "_per_", -1)
	resultMetricName = strings.ToLower(resultMetricName)
	return resultMetricName
}
