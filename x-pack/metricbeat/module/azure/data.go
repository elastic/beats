// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	// NoDimension is used to group metrics in separate api calls in order to reduce the number of executions
	NoDimension           = "none"
	nativeMetricset       = "monitor"
	replaceUpperCaseRegex = `(?:[^A-Z_\W])([A-Z])[^A-Z]`
)

// KeyValuePoint is a key/value point that represents a single metric value
// at a given timestamp.
//
// It also contains the metric dimensions and important metadata (resource ID,
// resource type, etc.).
type KeyValuePoint struct {
	Key           string
	Value         interface{}
	Namespace     string
	ResourceId    string
	ResourceSubId string
	Dimensions    mapstr.M
	TimeGrain     string
	Timestamp     time.Time
}

// mapToKeyValuePoints maps a list of `azure.Metric` to a list of `azure.KeyValuePoint`.
//
// `azure.KeyValuePoint` struct makes grouping metrics by timestamp, dimensions,
// and other fields more straightforward than using `azure.Metric`.
//
// How?
//
// `azure.Metric` has the following structure (simplified):
//
//	{
//	    "namespace": "Microsoft.Compute/virtualMachineScaleSets",
//	    "resource_id": "/subscriptions/123/resourceGroups/ABC/providers/Microsoft.Compute/virtualMachineScaleSets/aks-agentpool-12628255-vmss",
//	    "time_grain": "PT5M",
//	    "aggregations": "Total",
//	    "dimensions": [
//	        {
//	            "name": "VMName",
//	            "displayName": "*"
//	        }
//	    ],
//	    "names": [
//	        "Network In",
//	        "Network Out"
//	    ],
//	    "values": [
//	        {
//	            "name": "Network In",
//	            "timestamp": "2021-03-04T14:00:00Z",
//	            "total": 4211652,
//	            "dimensions": [
//	                {
//	                    "name": "VMName",
//	                    "value": "aks-agentpool-12628255-vmss_0"
//	                }
//	            ]
//	        },
//	        {
//	            "name": "Network Out",
//	            "timestamp": "2021-03-04T14:00:00Z",
//	            "total": 1105888,
//	            "dimensions": [
//	                {
//	                    "name": "VMName",
//	                    "value": "aks-agentpool-12628255-vmss_0"
//	                }
//	            ]
//	        }
//	    ]
//	}
//
// Here we have two metric values: "Network In" and "Network Out". Each metric value
// has a timestamp, a total, and a list of dimensions.
//
// To group the metrics using the `azure.Metric` structure, we need to assume that
// all metric values have the same timestamp and dimensions. This seems true during
// our tests, but I'm not 100% sure this is always the case.
//
// The alternative is to unpack the metric values into a list of `azure.KeyValuePoint`.
//
// The `mapToKeyValuePoints` function turns the previous `azure.Metric` in the a
// `azure.KeyValuePoint` list with the following structure (simplified):
//
// [
//
//	{
//	    "key": "network_in_total.total",
//	    "value": 4211652,
//	    "namespace": "Microsoft.Compute/virtualMachineScaleSets",
//	    "resource_id": "/subscriptions/123/resourceGroups/ABC/providers/Microsoft.Compute/virtualMachineScaleSets/aks-agentpool-12628255-vmss",
//	    "time_grain": "PT5M",
//	    "dimensions": {
//	        "VMName": "aks-agentpool-12628255-vmss_0"
//	    },
//	    "time": "2021-03-04T14:00:00Z"
//	},
//	{
//	    "key": "network_out_total.total",
//	    "value": 1105888,
//	    "namespace": "Microsoft.Compute/virtualMachineScaleSets",
//	    "resource_id": "/subscriptions/123/resourceGroups/ABC/providers/Microsoft.Compute/virtualMachineScaleSets/aks-agentpool-12628255-vmss",
//	    "time_grain": "PT5M",
//	    "dimensions": {
//	        "VMName": "aks-agentpool-12628255-vmss_0"
//	    },
//	    "time": "2021-03-04T14:00:00Z"
//	}
//
// ]
//
// With this structure, we can group the metrics by timestamp, dimensions, and
// other fields without making assumptions.
func mapToKeyValuePoints(metrics []Metric) []KeyValuePoint {
	var points []KeyValuePoint
	for _, metric := range metrics {
		for _, value := range metric.Values {
			metricName := managePropertyName(value.name)
			dimensions := mapstr.M{}
			if len(metric.Dimensions) == len(value.dimensions) {
				// Take the dimension name from the metric definition and the
				// dimension value from the metric value.
				//
				// Why?
				//
				// Because the dimension name in the metric value
				// comes in lower case.
				for _, dim := range metric.Dimensions {
					// Dimensions from metric definition and metric value are
					// not guaranteed to be in the same order, so we need to
					// find by name the right value for each dimension.
					// _, _ = point.Dimensions.Put(dim.Name, getDimensionValue(dim.Name, value.dimensions))
					_, _ = dimensions.Put(dim.Name, getDimensionValue(dim.Name, value.dimensions))
				}
			}

			if value.min != nil {
				points = append(points, KeyValuePoint{
					Key:           fmt.Sprintf("%s.%s", metricName, "min"),
					Value:         value.min,
					Namespace:     metric.Namespace,
					ResourceId:    metric.ResourceId,
					ResourceSubId: metric.ResourceSubId,
					TimeGrain:     metric.TimeGrain,
					Dimensions:    dimensions,
					Timestamp:     value.timestamp,
				})
			}

			if value.max != nil {
				points = append(points, KeyValuePoint{
					Key:           fmt.Sprintf("%s.%s", metricName, "max"),
					Value:         value.max,
					Namespace:     metric.Namespace,
					ResourceId:    metric.ResourceId,
					ResourceSubId: metric.ResourceSubId,
					TimeGrain:     metric.TimeGrain,
					Dimensions:    dimensions,
					Timestamp:     value.timestamp,
				})
			}

			if value.avg != nil {
				points = append(points, KeyValuePoint{
					Key:           fmt.Sprintf("%s.%s", metricName, "avg"),
					Value:         value.avg,
					Namespace:     metric.Namespace,
					ResourceId:    metric.ResourceId,
					ResourceSubId: metric.ResourceSubId,
					TimeGrain:     metric.TimeGrain,
					Dimensions:    dimensions,
					Timestamp:     value.timestamp,
				})
			}

			if value.total != nil {
				points = append(points, KeyValuePoint{
					Key:           fmt.Sprintf("%s.%s", metricName, "total"),
					Value:         value.total,
					Namespace:     metric.Namespace,
					ResourceId:    metric.ResourceId,
					ResourceSubId: metric.ResourceSubId,
					TimeGrain:     metric.TimeGrain,
					Dimensions:    dimensions,
					Timestamp:     value.timestamp,
				})
			}

			if value.count != nil {
				points = append(points, KeyValuePoint{
					Key:           fmt.Sprintf("%s.%s", metricName, "count"),
					Value:         value.count,
					Namespace:     metric.Namespace,
					ResourceId:    metric.ResourceId,
					ResourceSubId: metric.ResourceSubId,
					TimeGrain:     metric.TimeGrain,
					Dimensions:    dimensions,
					Timestamp:     value.timestamp,
				})
			}
		}
	}

	return points
}

// mapToEvents maps the metric values to events and reports them to Elasticsearch.
func mapToEvents(metrics []Metric, client *Client, reporter mb.ReporterV2) error {
	// Map the metric values into a list of key/value points.
	//
	// This makes it easier to group the metrics by timestamp
	// and dimensions.
	points := mapToKeyValuePoints(metrics)

	// Group the points by timestamp and other fields we consider
	// as dimensions for the whole event.
	//
	// Metrics have their own dimensions, and this is fine at the
	// metric level.
	//
	// We identified a set of field we consider as dimensions
	// at the event level. The event level dimensions define
	// the time series when TSDB is enabled.
	groupedPoints := make(map[string][]KeyValuePoint)
	for _, point := range points {
		groupingKey := fmt.Sprintf(
			"%s,%s,%s,%s,%s,%s",
			point.Timestamp,
			point.Namespace,
			point.ResourceId,
			point.ResourceSubId,
			point.Dimensions,
			point.TimeGrain,
		)

		groupedPoints[groupingKey] = append(groupedPoints[groupingKey], point)
	}

	// Create an event for each group of points and send
	// it to Elasticsearch.
	for _, _points := range groupedPoints {
		if len(_points) == 0 {
			// This should never happen, but I don't feel like
			// writing points[0] without checking the length first.
			continue
		}

		// We assume that all points have the same timestamp and
		// dimensions because they were grouped by the same key.
		//
		// We use the reference point to get the resource ID and
		// all other information common to all points.
		referencePoint := _points[0]

		// Look up the full cloud resource information in the cache.
		resource := client.LookupResource(referencePoint.ResourceId)

		// Build the event using all the information we have.
		event, err := buildEventFrom(referencePoint, _points, resource, client.Config.DefaultResourceType)
		if err != nil {
			return err
		}

		//
		// Enrich the event with cloud metadata.
		//
		if client.Config.AddCloudMetadata {
			vm := client.GetVMForMetadata(&resource, referencePoint)
			addCloudVMMetadata(&event, vm, resource.Subscription)
		}

		//
		// Report the event to Elasticsearch.
		//
		reporter.Event(event)
	}

	return nil
}

// buildEventFrom build an event from a group of points.
func buildEventFrom(referencePoint KeyValuePoint, points []KeyValuePoint, resource Resource, defaultResourceType string) (mb.Event, error) {
	event := mb.Event{
		ModuleFields: mapstr.M{
			"timegrain": referencePoint.TimeGrain,
			"namespace": referencePoint.Namespace,
			"resource": mapstr.M{
				"type":  resource.Type,
				"group": resource.Group,
				"name":  resource.Name,
			},
			"subscription_id": resource.Subscription,
		},
		MetricSetFields: mapstr.M{},
		Timestamp:       referencePoint.Timestamp,
		RootFields: mapstr.M{
			"cloud": mapstr.M{
				"provider": "azure",
				"region":   resource.Location,
			},
		},
	}

	if referencePoint.ResourceSubId != "" {
		_, _ = event.ModuleFields.Put("resource.id", referencePoint.ResourceSubId)
	} else {
		_, _ = event.ModuleFields.Put("resource.id", resource.Id)
	}
	if len(resource.Tags) > 0 {
		_, _ = event.ModuleFields.Put("resource.tags", resource.Tags)
	}

	if len(referencePoint.Dimensions) > 0 {
		for key, value := range referencePoint.Dimensions {
			if value == "*" {
				_, _ = event.ModuleFields.Put(fmt.Sprintf("dimensions.%s", managePropertyName(key)), getDimensionValueForKeyValuePoint(key, referencePoint.Dimensions))
			} else {
				_, _ = event.ModuleFields.Put(fmt.Sprintf("dimensions.%s", managePropertyName(key)), value)
			}
		}
	}

	metricList := mapstr.M{}
	for _, point := range points {
		_, _ = metricList.Put(point.Key, point.Value)
	}

	// I don't know why we are doing it, but we need to keep it
	// for now for backwards compatibility.
	//
	// There are Metricbeat modules and Elastic Agent integrations
	// that rely on this.
	if defaultResourceType == "" {
		_, _ = event.ModuleFields.Put("metrics", metricList)
	} else {
		for key, metric := range metricList {
			_, _ = event.MetricSetFields.Put(key, metric)
		}
	}

	// Enrich the event with host metadata.
	addHostMetadata(&event, metricList)

	return event, nil
}

// managePropertyName function will handle metric names, there are several formats the metric names are written
func managePropertyName(metric string) string {
	// replace spaces with underscores
	resultMetricName := strings.Replace(metric, " ", "_", -1)
	// replace backslashes with "per"
	resultMetricName = strings.Replace(resultMetricName, "/", "_per_", -1)
	resultMetricName = strings.Replace(resultMetricName, "\\", "_", -1)
	// replace actual percentage symbol with the smbol "pct"
	resultMetricName = strings.Replace(resultMetricName, "_%_", "_pct_", -1)
	// create an object in case of ":"
	resultMetricName = strings.Replace(resultMetricName, ":", "_", -1)
	// create an object in case of ":"
	resultMetricName = strings.Replace(resultMetricName, "_-_", "_", -1)
	// replace uppercases with underscores
	resultMetricName = ReplaceUpperCase(resultMetricName)

	//  avoid cases as this "logicaldisk_avg._disk_sec_per_transfer"
	obj := strings.Split(resultMetricName, ".")
	for index := range obj {
		// in some cases a trailing "_" is found
		obj[index] = strings.TrimPrefix(obj[index], "_")
		obj[index] = strings.TrimSuffix(obj[index], "_")
	}
	resultMetricName = strings.ToLower(strings.Join(obj, "_"))

	return resultMetricName
}

// ReplaceUpperCase func will replace upper case with '_'
func ReplaceUpperCase(src string) string {
	replaceUpperCaseRegexp := regexp.MustCompile(replaceUpperCaseRegex)
	return replaceUpperCaseRegexp.ReplaceAllStringFunc(src, func(str string) string {
		var newStr string
		for _, r := range str {
			// split into fields based on class of unicode character
			if unicode.IsUpper(r) {
				newStr += "_" + strings.ToLower(string(r))
			} else {
				newStr += string(r)
			}
		}
		return newStr
	})
}

// getDimensionValue will return dimension value for the key provided
func getDimensionValue(dimension string, dimensions []Dimension) string {
	for _, dim := range dimensions {
		if strings.EqualFold(dim.Name, dimension) {
			return dim.Value
		}
	}

	return ""
}

// getDimensionValue2 will return dimension value for the key provided
func getDimensionValueForKeyValuePoint(dimension string, dimensions mapstr.M) string {
	for key, value := range dimensions {
		if strings.EqualFold(key, dimension) {
			return fmt.Sprintf("%v", value)
		}
	}

	return ""
}
