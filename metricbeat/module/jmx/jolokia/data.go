package jolokia

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

type Entry struct {
	Request struct {
		Mbean string `json:"mbean"`
	}
	Value map[string]interface{}
}

// Map responseBody to common.MapStr
func eventMapping(responseBody []byte, mapping map[string]string, application string, instance string) (common.MapStr, error) {

	debugf("Got reponse body: ", string(responseBody[:]))

	var entries []Entry
	err := json.Unmarshal(responseBody, &entries)

	if err != nil {
		return nil, fmt.Errorf("Cannot unmarshal json response: %s", err)
	}

	debugf("Unmarshalled json: ", entries)

	event := map[string]interface{}{}

	if application != "" {
		event["application"] = application
	}

	if instance != "" {
		event["instance"] = instance
	}

	for _, v := range entries {
		for attribute, value := range v.Value {
			event = parseSingleResponse(v.Request.Mbean, attribute, value, event, mapping)
		}
	}

	return event, nil

}

// getValue finds a value in a map and passes it back
func getValue(key string, data map[string]string) (string, error) {
	value, exists := data[key]

	if !exists {
		return "", fmt.Errorf("Key `%s` not found", key)
	}

	return value, nil
}

func parseSingleResponse(mbeanName string, attributeName string, attibuteValue interface{}, event map[string]interface{}, mapping map[string]string) map[string]interface{} {
	//create metric name by merging mbean and attribute fields
	var metricName = mbeanName + "_" + attributeName
	//find alias for the metric
	var alias, err = getValue(metricName, mapping)
	if err != nil {
		debugf("No alias found for metric: '%s', skipping...", metricName)
		return event
	}

	debugf("metricName: '%s'", metricName)
	debugf("alias: '%s'", alias)
	//split alias by dot to check if it`s a nested value
	aliasStructure := strings.Split(alias, ".")
	switch len(aliasStructure) {
	case 1:
		//check if node already exists
		if node, exists := event[aliasStructure[0]]; exists {
			debugf("The alias '%s' already exists and won`t be overwritten", node)
			return event
		}
		event[aliasStructure[0]] = attibuteValue
	case 2:
		//check if node already exists to avoid overwriting it
		if nested, exists := event[aliasStructure[0]]; exists {
			if nested, ok := nested.(map[string]interface{}); ok {
				//add to existing map
				nested[aliasStructure[1]] = attibuteValue
			} else {
				debugf("The alias '%s' already exists and is not nested, skipping...", aliasStructure[0])
			}
		} else {
			//init map and add value
			event[aliasStructure[0]] = map[string]interface{}{aliasStructure[1]: attibuteValue}
		}
	default:
		_ = fmt.Errorf("Mapping failed, alias nesting depth exceeds 1: %d", len(aliasStructure)-1)
	}

	return event
}
