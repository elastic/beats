package util

import (
	"github.com/elastic/beats/libbeat/common"
)

// GetString from the given event, "" if it doesn't exist
func GetString(m common.MapStr, field string) string {
	val, err := m.GetValue(field)
	if err != nil {
		return ""
	}

	valStr, ok := val.(string)
	if !ok {
		return ""
	}
	return valStr
}

// GetFloat64 from the given event, 0 if it doesn't exist
func GetFloat64(m common.MapStr, field string) float64 {
	val, err := m.GetValue(field)
	if err != nil {
		return 0
	}

	valStr, ok := val.(float64)
	if !ok {
		return 0
	}
	return valStr
}

// GetInt64 from the given event, 0 if it doesn't exist
func GetInt64(m common.MapStr, field string) int64 {
	val, err := m.GetValue(field)
	if err != nil {
		return 0
	}

	valStr, ok := val.(int64)
	if !ok {
		return 0
	}
	return valStr
}

// MergeEvents from b events into a. The process will ensure that:
//   - only events matching the given filter are processed.
//   - fields in the delete list will be removed from the event
//   - match fields will be used to match events from a & b, if all fields are equal, they will be merged
func MergeEvents(a, b []common.MapStr, filter map[string]string, delete []string, match []string) []common.MapStr {
	events := map[string]common.MapStr{}
	for _, event := range a {
		id := eventID(event, match)
		events[id] = event
	}

	for _, event := range b {
		// Skip events that don't match the filter
		skip := false
		for k, v := range filter {
			if GetString(event, k) != v {
				skip = true
			}
		}
		if skip {
			continue
		}

		for _, field := range delete {
			event.Delete(field)
		}

		id := eventID(event, match)
		if data, ok := events[id]; ok {
			data.DeepUpdate(event)
		} else {
			events[id] = event
		}
	}

	result := []common.MapStr{}
	for _, e := range events {
		result = append(result, e)
	}

	return result
}

func eventID(event common.MapStr, fields []string) string {
	var id string
	for _, field := range fields {
		id += GetString(event, field) + "-"
	}
	return id
}
