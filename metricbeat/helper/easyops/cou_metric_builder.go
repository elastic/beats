package easyops

import (
	"fmt"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"strings"
)

type couMetricBuilder struct {
	baseBuilderFields
}

func newCouMetricBuilder(field string, originMetric []string, groupKeys []string, defaultValues map[string]interface{}) AggregateMetricBuilder {
	return &couMetricBuilder{
		baseBuilderFields{
			field:         field,
			originMetrics: originMetric,
			groupKeys:     groupKeys,
			defaultValues: defaultValues,
		},
	}
}

func (builder *couMetricBuilder) Build(events []mapstr.M) []mapstr.M {
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
		counters := builder.count(es, builder.originMetrics, builder.defaultValues)
		for val, count := range counters {
			field := strings.Replace(builder.field, "{}", val, 1)
			_, _ = rs.Put(field, count)
		}
		result = append(result, rs)
	}
	return result
}

func (builder *couMetricBuilder) count(events []mapstr.M, originMetric []string, defaultValues map[string]interface{}) map[string]float64 {
	counters := map[string]float64{}
	for field, defaultValue := range defaultValues {
		if value, ok := defaultValue.(float64); ok {
			counters[field] = value
		}
	}
	for _, metric := range originMetric {
		for _, event := range events {
			value, err := event.GetValue(metric)
			if err == nil {
				val := strings.ToLower(fmt.Sprintf("%v", value))
				if _, ok := counters[val]; !ok {
					counters[val] = 0
				}
				counters[val] += 1
			}
		}
	}
	return counters
}
