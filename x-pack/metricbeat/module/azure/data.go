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
			point := KeyValuePoint{
				Timestamp:  value.timestamp,
				Dimensions: mapstr.M{},
			}

			metricName := managePropertyName(value.name)
			switch {
			case value.min != nil:
				point.Key = fmt.Sprintf("%s.%s", metricName, "min")
				point.Value = value.avg
			case value.max != nil:
				point.Key = fmt.Sprintf("%s.%s", metricName, "max")
				point.Value = value.avg
			case value.avg != nil:
				point.Key = fmt.Sprintf("%s.%s", metricName, "avg")
				point.Value = value.avg
			case value.total != nil:
				point.Key = fmt.Sprintf("%s.%s", metricName, "total")
				point.Value = value.total
			case value.count != nil:
				point.Key = fmt.Sprintf("%s.%s", metricName, "count")
				point.Value = value.count
			}

			point.Namespace = metric.Namespace
			point.ResourceId = metric.ResourceId
			point.ResourceSubId = metric.ResourceSubId
			point.TimeGrain = metric.TimeGrain

			// The number of dimensions in the metric definition and the
			// number of dimensions in the metric values should be the same.
			//
			// But, since definitions and values are retrieved from different
			// API endpoints, we need to make sure that we don't panic if the
			// number of dimensions is different.
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
					_, _ = point.Dimensions.Put(dim.Name, getDimensionValue(dim.Name, value.dimensions))
				}
			}

			points = append(points, point)
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

//// mapToEvents will map metric values to beats events
//func mapToEvents(metrics []Metric, client *Client, report mb.ReporterV2) error {
//	// metrics and metric values are currently grouped relevant to the azure REST API calls (metrics with the same aggregations per call)
//	// multiple metrics can be mapped in one event depending on the resource, namespace, dimensions and timestamp
//
//	// grouping metrics by resource and namespace
//	groupByResourceNamespace := make(map[string][]Metric)
//	for _, metric := range metrics {
//		// Skip metrics with no values
//		if len(metric.Values) == 0 {
//			continue
//		}
//		// build a resource key with unique resource namespace combination
//		groupingKey := fmt.Sprintf("%s,%s", metric.ResourceId, metric.Namespace)
//		groupByResourceNamespace[groupingKey] = append(groupByResourceNamespace[groupingKey], metric)
//	}
//
//	// grouping metrics by the dimensions configured
//	groupByDimensions := make(map[string][]Metric)
//	for resNamKey, resourceMetrics := range groupByResourceNamespace {
//		for _, resourceMetric := range resourceMetrics {
//			if len(resourceMetric.Dimensions) == 0 {
//				groupByDimensions[resNamKey+NoDimension] = append(groupByDimensions[resNamKey+NoDimension], resourceMetric)
//			} else {
//				var dimKey string
//				for _, dim := range resourceMetric.Dimensions {
//					dimKey += dim.Name + dim.Value
//				}
//				groupByDimensions[resNamKey+dimKey] = append(groupByDimensions[resNamKey+dimKey], resourceMetric)
//			}
//		}
//	}
//
//	// Grouping metric values by timestamp and creating events
//	// (for each metric the REST api can retrieve multiple metric values
//	// for same aggregation  but different timeframes).
//	for _, grouped := range groupByDimensions {
//
//		defaultMetric := grouped[0]
//		resource := client.GetResourceForMetaData(defaultMetric)
//		groupByTimeMetrics := make(map[time.Time][]MetricValue)
//
//		for _, metric := range grouped {
//			for _, m := range metric.Values {
//				groupByTimeMetrics[m.timestamp] = append(groupByTimeMetrics[m.timestamp], m)
//			}
//		}
//
//		for timestamp, groupTimeValues := range groupByTimeMetrics {
//			var event mb.Event
//			var metricList mapstr.M
//			var vm VmResource
//
//			// group events by dimension values
//			exists, validDimensions := returnAllDimensions(defaultMetric.Dimensions)
//			if exists {
//				for _, selectedDimension := range validDimensions {
//					groupByDimensions := make(map[string][]MetricValue)
//					for _, dimGroupValue := range groupTimeValues {
//						dimKey := fmt.Sprintf("%s,%s", selectedDimension.Name, getDimensionValue(selectedDimension.Name, dimGroupValue.dimensions))
//						groupByDimensions[dimKey] = append(groupByDimensions[dimKey], dimGroupValue)
//					}
//
//					for _, groupDimValues := range groupByDimensions {
//						manageAndReportEvent(client, report, event, metricList, vm, timestamp, defaultMetric, resource, groupDimValues)
//					}
//				}
//			} else {
//				manageAndReportEvent(client, report, event, metricList, vm, timestamp, defaultMetric, resource, groupTimeValues)
//			}
//		}
//	}
//
//	return nil
//}

//// manageAndReportEvent function will handle event creation and report
//func manageAndReportEvent(client *Client, report mb.ReporterV2, event mb.Event, metricList mapstr.M, vm VmResource, timestamp time.Time, defaultMetric Metric, resource Resource, groupedValues []MetricValue) {
//	event, metricList = createEvent(timestamp, defaultMetric, resource, groupedValues)
//	//if client.Config.AddCloudMetadata {
//	//	vm = client.GetVMForMetadata(&resource, groupedValues)
//	//	addCloudVMMetadata(&event, vm, resource.Subscription)
//	//}
//
//	if client.Config.DefaultResourceType == "" {
//		event.ModuleFields.Put("metrics", metricList)
//	} else {
//		for key, metric := range metricList {
//			event.MetricSetFields.Put(key, metric)
//		}
//	}
//
//	report.Event(event)
//}

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

//// createEvent will create a new base event
//func createEvent(timestamp time.Time, metric Metric, resource Resource, metricValues []MetricValue) (mb.Event, mapstr.M) {
//
//	event := mb.Event{
//		ModuleFields: mapstr.M{
//			"timegrain": metric.TimeGrain,
//			"namespace": metric.Namespace,
//			"resource": mapstr.M{
//				"type":  resource.Type,
//				"group": resource.Group,
//				"name":  resource.Name,
//			},
//			"subscription_id": resource.Subscription,
//		},
//		MetricSetFields: mapstr.M{},
//		Timestamp:       timestamp,
//		RootFields: mapstr.M{
//			"cloud": mapstr.M{
//				"provider": "azure",
//				"region":   resource.Location,
//			},
//		},
//	}
//	if metric.ResourceSubId != "" {
//		event.ModuleFields.Put("resource.id", metric.ResourceSubId)
//	} else {
//		event.ModuleFields.Put("resource.id", resource.Id)
//	}
//	if len(resource.Tags) > 0 {
//		event.ModuleFields.Put("resource.tags", resource.Tags)
//	}
//
//	if len(metric.Dimensions) > 0 {
//		for _, dimension := range metric.Dimensions {
//			if dimension.Value == "*" {
//				event.ModuleFields.Put(fmt.Sprintf("dimensions.%s", managePropertyName(dimension.Name)), getDimensionValue(dimension.Name, metricValues[0].dimensions))
//			} else {
//				event.ModuleFields.Put(fmt.Sprintf("dimensions.%s", managePropertyName(dimension.Name)), dimension.Value)
//			}
//
//		}
//	}
//
//	metricList := mapstr.M{}
//	for _, value := range metricValues {
//		metricNameString := fmt.Sprintf("%s", managePropertyName(value.name))
//		if value.min != nil {
//			metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "min"), *value.min)
//		}
//		if value.max != nil {
//			metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "max"), *value.max)
//		}
//		if value.avg != nil {
//			metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "avg"), *value.avg)
//		}
//		if value.total != nil {
//			metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "total"), *value.total)
//		}
//		if value.count != nil {
//			metricList.Put(fmt.Sprintf("%s.%s", metricNameString, "count"), *value.count)
//		}
//	}
//	addHostMetadata(&event, metricList)
//
//	return event, metricList
//}

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

//// returnAllDimensions will check if users has entered a filter for all dimension values (*)
//func returnAllDimensions(dimensions []Dimension) (bool, []Dimension) {
//	if len(dimensions) == 0 {
//		return false, nil
//	}
//	var dims []Dimension
//	for _, dim := range dimensions {
//		if dim.Value == "*" {
//			dims = append(dims, dim)
//		}
//	}
//	if len(dims) == 0 {
//		return false, nil
//	}
//	return true, dims
//}
