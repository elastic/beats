package easyops

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common"
)

func GenerateGroupValue(event common.MapStr, groupKeys []string) (string, error) {
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

func GroupEventsByKeys(events []common.MapStr, groupKeys []string) map[string][]common.MapStr {
	result := map[string][]common.MapStr{}
	for _, event := range events {
		group, err := GenerateGroupValue(event, groupKeys)
		if err == nil {
			result[group] = append(result[group], event)
		}
	}
	return result
}
