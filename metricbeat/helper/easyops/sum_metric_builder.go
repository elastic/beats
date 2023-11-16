package easyops

import "github.com/elastic/elastic-agent-libs/mapstr"

type sumMetricBuilder struct {
	baseBuilderFields
}

func newSumMetricBuilder(field string, originMetric []string, groupKeys []string, defaultValues map[string]interface{}) AggregateMetricBuilder {
	return &sumMetricBuilder{
		baseBuilderFields{
			field:         field,
			originMetrics: originMetric,
			groupKeys:     groupKeys,
			defaultValues: defaultValues,
		},
	}
}

func (builder *sumMetricBuilder) Build(events []mapstr.M) []mapstr.M {
	var result []mapstr.M
	eventMap := GroupEventsByKeys(events, builder.groupKeys)
	for _, es := range eventMap {
		if len(es) == 0 {
			continue
		}
		rs := mapstr.M{}
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

func (builder *sumMetricBuilder) sum(events []mapstr.M, originMetric []string) interface{} {
	var floatResult float64 = 0
	for _, metric := range originMetric {
		for _, event := range events {
			value, err := event.GetValue(metric)
			if err == nil {
				floatResult += ConvertNumericValue(value)
			}
		}
	}
	return floatResult
}
