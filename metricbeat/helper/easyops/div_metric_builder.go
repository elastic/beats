package easyops

import (
	"math"

	"github.com/elastic/beats/v7/libbeat/common"
)

type divMetricBuilder struct {
	baseBuilderFields
}

func newDivMetricBuilder(field string, originMetric []string, groupKeys []string) AggregateMetricBuilder {
	return &divMetricBuilder{
		baseBuilderFields{
			field:         field,
			originMetrics: originMetric,
			groupKeys:     groupKeys,
		},
	}
}

func (builder *divMetricBuilder) Build(events []common.MapStr) []common.MapStr {
	var result []common.MapStr
	eventMap := GroupEventsByKeys(events, builder.groupKeys)
	for _, es := range eventMap {
		if len(es) == 0 {
			continue
		}
		rs := common.MapStr{}
		for _, groupKey := range builder.groupKeys {
			// GetValue success in GroupEventsByKeys
			val, _ := es[0].GetValue(groupKey)
			_, _ = rs.Put(groupKey, val)
		}
		_, _ = rs.Put(builder.field, builder.div(es, builder.originMetrics))
		result = append(result, rs)
	}
	return result
}

func (builder *divMetricBuilder) div(events []common.MapStr, originMetric []string) interface{} {
	var floatResult float64 = 0
MetricLoop:
	for index, metric := range originMetric {
		for _, event := range events {
			value, err := event.GetValue(metric)
			if err == nil {
				var val float64 = 0
				switch value.(type) {
				case int:
					val = float64(value.(int))
				case int8:
					val = float64(value.(int8))
				case int16:
					val = float64(value.(int16))
				case int32:
					val = float64(value.(int32))
				case int64:
					val = float64(value.(int64))
				case uint:
					val = float64(value.(uint))
				case uint8:
					val = float64(value.(uint8))
				case uint16:
					val = float64(value.(uint16))
				case uint32:
					val = float64(value.(uint32))
				case uint64:
					val = float64(value.(uint64))
				case float32:
					val = float64(value.(float32))
				case float64:
					val = float64(value.(float64))
				}
				if index == 0 {
					floatResult = val
				} else if val != 0 {
					floatResult /= val
				} else {
					floatResult = math.MaxFloat64
					break MetricLoop
				}
			}
		}
	}
	return floatResult
}
