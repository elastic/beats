package collector

import (
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

type PromEvent struct {
	key       string
	value     interface{}
	labels    common.MapStr
	labelHash string
}

// NewPromEvent creates a prometheus event based on the given string
func NewPromEvent(line string) PromEvent {
	// Separate key and value
	splitPos := strings.LastIndex(line, " ")
	split := []string{line[:splitPos], line[splitPos+1:]}

	promEvent := PromEvent{
		key:       split[0],
		labelHash: "_", // _ represents empty labels
	}

	// skip entries without a value
	if split[1] == "NaN" {
		promEvent.value = nil
	} else {
		promEvent.value = convertValue(split[1])
	}

	// Split key
	startLabels := strings.Index(line, "{")
	endLabels := strings.Index(line, "}")

	// Handle labels
	if startLabels != -1 {
		// Overwrite key, as key contained labels until now too
		promEvent.key = line[0:startLabels]
		promEvent.labelHash = line[startLabels+1 : endLabels]
		// Extract labels
		promEvent.labels = extractLabels(promEvent.labelHash)
	}

	return promEvent
}

// extractLabels splits up a label string of format handler="alerts",quantile="0.5"
// into a key / value list
func extractLabels(labelsString string) common.MapStr {

	keyValuePairs := common.MapStr{}

	// Extract labels
	labels := strings.Split(labelsString, "\",")
	for _, label := range labels {
		keyValue := strings.Split(label, "=")
		// Remove " from value
		keyValue[1] = strings.Trim(keyValue[1], "\"")

		// Converts value to int or float if needed
		keyValuePairs[keyValue[0]] = convertValue(keyValue[1])
	}

	return keyValuePairs
}

// convertValue takes the input string and converts it to int of float
func convertValue(value string) interface{} {

	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return i
	}

	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}

	return value
}
