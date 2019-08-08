package monitor

import (
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"strings"
	"time"
)

func filterMetrics(selectedRange []string, allRange insights.MetricDefinitionCollection) ([]string, []string) {
	var inRange []string
	var notInRange []string
	var allMetrics []string
	for _, definition := range *allRange.Value {
		allMetrics = append(allMetrics, *definition.Name.Value)
	}
	for _, name := range selectedRange {
		if stringInSlice(name, allMetrics) {
			inRange = append(inRange, name)
		} else {
			notInRange = append(notInRange, name)
		}

	}
	return inRange, notInRange
}

func filterAggregations(selectedRange []string, metrics []insights.MetricDefinition) ([]string, []string) {
	var supported []string
	var unsupported []string
	for _, metric := range metrics {
		for _, agg := range *metric.SupportedAggregationTypes {
			supported = append(supported, string(agg))
		}
		selectedRange, unsupported = intersections(supported, selectedRange)
	}
	return selectedRange, unsupported
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
func filter(src []string) (res []string) {
	for _, s := range src {
		newStr := strings.Join(res, " ")
		if !strings.Contains(newStr, s) {
			res = append(res, s)
		}
	}
	return
}

func intersections(section1, section2 []string) (intersection []string, difference []string) {
	str1 := strings.Join(filter(section1), " ")
	for _, s := range filter(section2) {
		if strings.Contains(str1, s) {
			intersection = append(intersection, s)
		} else {
			difference = append(difference, s)
		}
	}
	return
}

func getMetricDefinitionsByNames(metricDefs insights.MetricDefinitionCollection, names []string) []insights.MetricDefinition {
	var metrics []insights.MetricDefinition
	for _, def := range *metricDefs.Value {
		for _, supportedName := range names {
			if *def.Name.Value == supportedName {
				metrics = append(metrics, def)
			}
		}
	}
	return metrics
}

func (p *ResourceConfiguration) expired() bool {
	if p.refreshInterval <= 0 {
		return true
	}
	p.lastUpdate.Lock()
	defer p.lastUpdate.Unlock()
	if p.lastUpdate.Add(p.refreshInterval).After(time.Now()) {
		return false
	}
	p.lastUpdate.Time = time.Now()
	return true
}
