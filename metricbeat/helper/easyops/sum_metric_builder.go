package easyops

import "github.com/elastic/beats/v7/libbeat/common"

type sumMetricBuilder struct {
	baseBuilderFields
}

func newSumMetricBuilder(field string, originMetric []string, groupKeys []string) AggregateMetricBuilder {
	return &sumMetricBuilder{
		baseBuilderFields{
			field:         field,
			originMetrics: originMetric,
			groupKeys:     groupKeys,
		},
	}
}

func (builder *sumMetricBuilder) Build(events []common.MapStr) []common.MapStr {
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
		_, _ = rs.Put(builder.field, builder.sum(es, builder.originMetrics))
		result = append(result, rs)
	}
	return result
}

func (builder *sumMetricBuilder) sum(events []common.MapStr, originMetric []string) interface{} {
	var floatResult float64 = 0
	for _, metric := range originMetric {
		for _, event := range events {
			value, err := event.GetValue(metric)
			if err == nil {
				switch value.(type) {
				case int:
					floatResult += float64(value.(int))
				case int8:
					floatResult += float64(value.(int8))
				case int16:
					floatResult += float64(value.(int16))
				case int32:
					floatResult += float64(value.(int32))
				case int64:
					floatResult += float64(value.(int64))
				case uint:
					floatResult += float64(value.(uint))
				case uint8:
					floatResult += float64(value.(uint8))
				case uint16:
					floatResult += float64(value.(uint16))
				case uint32:
					floatResult += float64(value.(uint32))
				case uint64:
					floatResult += float64(value.(uint64))
				case float32:
					floatResult += float64(value.(float32))
				case float64:
					floatResult += float64(value.(float64))
				}
			}
		}
	}
	return floatResult
}
