package dashboards

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type JSONObjectAttribute struct {
	Description           string                 `json:"description"`
	KibanaSavedObjectMeta map[string]interface{} `json:"kibanaSavedObjectMeta"`
	Title                 string                 `json:"title"`
	Type                  string                 `json:"type"`
}

type JSONObject struct {
	Attributes JSONObjectAttribute `json:"attributes"`
}

type JSONFormat struct {
	Objects []JSONObject `json:"objects"`
}

func ReplaceIndexInIndexPattern(index string, content common.MapStr) common.MapStr {

	if index != "" {
		// change index pattern name
		if objects, ok := content["objects"].([]interface{}); ok {
			for i, object := range objects {
				if objectMap, ok := object.(map[string]interface{}); ok {
					objectMap["id"] = index

					if attributes, ok := objectMap["attributes"].(map[string]interface{}); ok {
						attributes["title"] = index
						objectMap["attributes"] = attributes
					}
					objects[i] = objectMap
				}
			}
			content["objects"] = objects
		}
	}

	return content
}

func replaceIndexInSearchObject(index string, savedObject string) (string, error) {

	var record common.MapStr
	err := json.Unmarshal([]byte(savedObject), &record)
	if err != nil {
		return "", fmt.Errorf("fail to unmarshal searchSourceJSON from search : %v", err)
	}

	if _, ok := record["index"]; ok {
		record["index"] = index
	}
	searchSourceJSON, err := json.Marshal(record)
	if err != nil {
		return "", fmt.Errorf("fail to marshal searchSourceJSON: %v", err)
	}

	return string(searchSourceJSON), nil
}

func ReplaceIndexInSavedObject(index string, kibanaSavedObject map[string]interface{}) map[string]interface{} {

	if searchSourceJSON, ok := kibanaSavedObject["searchSourceJSON"].(string); ok {
		searchSourceJSON, err := replaceIndexInSearchObject(index, searchSourceJSON)
		if err != nil {
			logp.Err("Fail to replace searchSourceJSON: %v", err)
			return kibanaSavedObject
		}
		kibanaSavedObject["searchSourceJSON"] = searchSourceJSON
	}

	return kibanaSavedObject
}

func ReplaceIndexInDashboardObject(index string, content common.MapStr) common.MapStr {

	if index == "" {
		return content
	}
	if objects, ok := content["objects"].([]interface{}); ok {
		for i, object := range objects {
			if objectMap, ok := object.(map[string]interface{}); ok {
				if attributes, ok := objectMap["attributes"].(map[string]interface{}); ok {

					if kibanaSavedObject, ok := attributes["kibanaSavedObjectMeta"].(map[string]interface{}); ok {

						attributes["kibanaSavedObjectMeta"] = ReplaceIndexInSavedObject(index, kibanaSavedObject)
					}

					objectMap["attributes"] = attributes
				}
				objects[i] = objectMap
			}
		}
		content["objects"] = objects
	}
	return content
}

func ReplaceStringInDashboard(old, new string, content common.MapStr) (common.MapStr, error) {
	marshaled, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("fail to marshal dashboard object: %v", content)
	}

	replaced := bytes.Replace(marshaled, []byte(old), []byte(new), -1)

	var result common.MapStr
	err = json.Unmarshal(replaced, &result)
	return result, nil
}
