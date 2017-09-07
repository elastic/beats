package collector

import (
	"encoding/json"
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

type DropWizardEvent struct {
	key     string
	value   common.MapStr
	tags    common.MapStr
	tagHash string
}

// NewPromEvent creates a prometheus event based on the given string
func eventMapping(metrics map[string]interface{}) map[string]common.MapStr {
	eventList := map[string]common.MapStr{}

	for _, metricSet := range metrics {
		switch t := metricSet.(type) {
		case map[string]interface{}:
			for key, value := range t {
				name, tags := splitTagsFromMetricName(key)
				valueMap := common.MapStr{}

				metric, _ := value.(map[string]interface{})
				for k, v := range metric {
					switch v.(type) {
					case string:
						valueMap[k] = v

					case json.Number:
						valueMap[k] = convertValue(v.(json.Number))
					}

				}

				dropEvent := DropWizardEvent{
					key:   name,
					value: valueMap,
				}

				if len(tags) != 0 {
					dropEvent.tags = tags
					dropEvent.tagHash = tags.String()
				} else {
					dropEvent.tagHash = "_"
				}

				if _, ok := eventList[dropEvent.tagHash]; !ok {
					eventList[dropEvent.tagHash] = common.MapStr{}

					// Add labels
					if len(dropEvent.tags) > 0 {
						eventList[dropEvent.tagHash]["tags"] = dropEvent.tags
					}

				}
				eventList[dropEvent.tagHash][dropEvent.key] = dropEvent.value

			}

		default:
			continue
		}

	}

	return eventList
}

func splitTagsFromMetricName(metricName string) (string, common.MapStr) {
	if metricName == "" {
		return "", nil
	}

	index := strings.Index(metricName, "{")
	if index == -1 {
		return metricName, nil
	}

	key := metricName[:index]
	tags := common.MapStr{}

	tagStr := metricName[index+1 : len(metricName)-1]

	for {
		dobreak := false
		ind := strings.Index(tagStr, ",")
		eqPos := strings.Index(tagStr, "=")
		if ind == -1 {
			tags[tagStr[:eqPos]] = tagStr[eqPos+1:]
			dobreak = true
		} else {
			tags[tagStr[:eqPos]] = tagStr[eqPos+1 : ind]
			tagStr = tagStr[ind+2:]
		}

		if dobreak == true {
			break
		}
	}

	return key, tags
}

// convertValue takes the input string and converts it to int of float
func convertValue(value json.Number) interface{} {
	if i, err := value.Int64(); err == nil {
		return i
	}

	if f, err := value.Float64(); err == nil {
		return f
	}

	return value.String()
}
