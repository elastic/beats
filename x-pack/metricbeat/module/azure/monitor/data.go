// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package monitor

import (
	"fmt"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"strings"
	"time"
)

// eventsMapping will map metric values to beats events
func eventsMapping(report mb.ReporterV2, metrics []Metric) error {
	for _, metric := range metrics {
		if len(metric.values) == 0 {
			continue
		}
		groupByTimeMetrics := make(map[time.Time][]MetricValue)
		for _, m := range metric.values {
			groupByTimeMetrics[m.timestamp] = append(groupByTimeMetrics[m.timestamp], m)
		}
		for timestamp, groupValue := range groupByTimeMetrics {
			metricList := common.MapStr{}
			for _, value := range groupValue {
				metricNameString := fmt.Sprintf("%s", managePropertyName(value.name))
				if value.min != nil {
					metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "min"), *value.min)
				}
				if value.max != nil {
					metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "max"), *value.max)
				}
				if value.average != nil {
					metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "avg"), *value.average)
				}
				if value.total != nil {
					metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "total"), *value.total)
				}
				if value.count != nil {
					metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "count"), *value.count)
				}
			}
			event := mb.Event{

				Timestamp: timestamp,
				MetricSetFields: common.MapStr{
					"resource": common.MapStr{
						"name": metric.resource.Name,
						"type": metric.resource.Type,
					},
					"namespace":      metric.namespace,
					"subscriptionID": "unique identifier",
					"metrics":        metricList,
				},
			}
			if len(metric.dimensions) > 0 {
				for _, dimension := range metric.dimensions {
					event.MetricSetFields.Put(fmt.Sprintf("dimensions.%s", managePropertyName(dimension.name)), dimension.value)
				}
			}
			event.RootFields = common.MapStr{}
			event.RootFields.Put("cloud.provider", "azure")
			event.RootFields.Put("cloud.region", metric.resource.Location)
			report.Event(event)
		}

	}

	return nil
}

func managePropertyName(metric string) string {
	resultMetricName := strings.Replace(metric, " ", "_", -1)
	resultMetricName = strings.Replace(resultMetricName, "/", "_per_", -1)
	resultMetricName = strings.ToLower(resultMetricName)
	return resultMetricName
}
