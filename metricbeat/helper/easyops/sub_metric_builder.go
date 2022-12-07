package easyops

import "github.com/elastic/beats/v7/libbeat/common"

type subMetricBuilder struct {
	baseBuilderFields
}

func newSubMetricBuilder(field string, originMetric []string, groupKeys []string, defaultValues map[string]interface{}) AggregateMetricBuilder {
	return &subMetricBuilder{
		baseBuilderFields{
			field:         field,
			originMetrics: originMetric,
			groupKeys:     groupKeys,
			defaultValues: defaultValues,
		},
	}
}

func (builder *subMetricBuilder) Build(events []common.MapStr) []common.MapStr {
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
		_, _ = rs.Put(builder.field, builder.sub(es, builder.originMetrics))
		result = append(result, rs)
	}
	return result
}

func (builder *subMetricBuilder) sub(events []common.MapStr, originMetric []string) interface{} {
	var floatResult float64 = 0
	for index, metric := range originMetric {
		for _, event := range events {
			value, err := event.GetValue(metric)
			if err == nil {
				val := ConvertNumericValue(value)
				if index == 0 {
					floatResult = val
				} else {
					floatResult -= val
				}
			}
		}
	}
	return floatResult
}
