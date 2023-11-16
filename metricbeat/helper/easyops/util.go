package easyops

import (
	"fmt"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"strings"
)

func GenerateGroupValue(event mapstr.M, groupKeys []string) (string, error) {
	groupValues := make([]string, 0, len(groupKeys))
	for _, groupKey := range groupKeys {
		value, err := event.GetValue(groupKey)
		if err != nil {
			return "", err
		}
		groupValues = append(groupValues, fmt.Sprintf("%s=%v", groupKey, value))
	}
	return strings.Join(groupValues, ";"), nil
}

func GroupEventsByKeys(events []mapstr.M, groupKeys []string) map[string][]mapstr.M {
	result := map[string][]mapstr.M{}
	for _, event := range events {
		group, err := GenerateGroupValue(event, groupKeys)
		if err == nil {
			result[group] = append(result[group], event)
		}
	}
	return result
}

func ConvertNumericValue(value interface{}) float64 {
	floatResult := 0.0
	switch value.(type) {
	case int:
		floatResult = float64(value.(int))
	case int8:
		floatResult = float64(value.(int8))
	case int16:
		floatResult = float64(value.(int16))
	case int32:
		floatResult = float64(value.(int32))
	case int64:
		floatResult = float64(value.(int64))
	case uint:
		floatResult = float64(value.(uint))
	case uint8:
		floatResult = float64(value.(uint8))
	case uint16:
		floatResult = float64(value.(uint16))
	case uint32:
		floatResult = float64(value.(uint32))
	case uint64:
		floatResult = float64(value.(uint64))
	case float32:
		floatResult = float64(value.(float32))
	case float64:
		floatResult = float64(value.(float64))
	}
	return floatResult
}
