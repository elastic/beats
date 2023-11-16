package easyops

import "github.com/elastic/elastic-agent-libs/mapstr"

type AggregateType uint8

var (
	AggregateTypeSum AggregateType = 0
	AggregateTypeDiv AggregateType = 1
	AggregateTypeSub AggregateType = 2
	AggregateTypeCou AggregateType = 3
)

type AggregateMetricMap struct {
	Type          AggregateType
	Field         string
	OriginMetrics []string
	GroupKeys     []string
	DefaultValues map[string]interface{}
}

type AggregateMetricBuilder interface {
	Build(events []mapstr.M) []mapstr.M
}

type baseBuilderFields struct {
	field         string
	originMetrics []string
	groupKeys     []string
	defaultValues map[string]interface{}
}

func NewAggregateMetricBuilder(metricMap AggregateMetricMap) AggregateMetricBuilder {
	switch metricMap.Type {
	case AggregateTypeSum:
		return newSumMetricBuilder(metricMap.Field, metricMap.OriginMetrics, metricMap.GroupKeys, metricMap.DefaultValues)
	case AggregateTypeDiv:
		return newDivMetricBuilder(metricMap.Field, metricMap.OriginMetrics, metricMap.GroupKeys, metricMap.DefaultValues)
	case AggregateTypeSub:
		return newSubMetricBuilder(metricMap.Field, metricMap.OriginMetrics, metricMap.GroupKeys, metricMap.DefaultValues)
	case AggregateTypeCou:
		return newCouMetricBuilder(metricMap.Field, metricMap.OriginMetrics, metricMap.GroupKeys, metricMap.DefaultValues)
	}
	return nil
}
