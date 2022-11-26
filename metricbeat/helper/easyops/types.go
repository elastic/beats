package easyops

import (
	"github.com/elastic/beats/v7/libbeat/common"
)

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
}

type AggregateMetricBuilder interface {
	Build(events []common.MapStr) []common.MapStr
}

type baseBuilderFields struct {
	field         string
	originMetrics []string
	groupKeys     []string
}

func NewAggregateMetricBuilder(metricMap AggregateMetricMap) AggregateMetricBuilder {
	switch metricMap.Type {
	case AggregateTypeSum:
		return newSumMetricBuilder(metricMap.Field, metricMap.OriginMetrics, metricMap.GroupKeys)
	case AggregateTypeDiv:
		return newDivMetricBuilder(metricMap.Field, metricMap.OriginMetrics, metricMap.GroupKeys)
	case AggregateTypeSub:
		return newSubMetricBuilder(metricMap.Field, metricMap.OriginMetrics, metricMap.GroupKeys)
	case AggregateTypeCou:
		return newCouMetricBuilder(metricMap.Field, metricMap.OriginMetrics, metricMap.GroupKeys)
	}
	return nil
}
