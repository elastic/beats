// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package dashboards

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

var (
	newline = []byte("\n")
)

// JSONObjectAttribute contains the attributes for a Kibana json object
type JSONObjectAttribute struct {
	Description           string                 `json:"description"`
	KibanaSavedObjectMeta map[string]interface{} `json:"kibanaSavedObjectMeta"`
	Title                 string                 `json:"title"`
	Type                  string                 `json:"type"`
	UiStateJSON           map[string]interface{} `json:"uiStateJSON"`
}

type JSONObject struct {
	Attributes JSONObjectAttribute `json:"attributes"`
}

type JSONFormat struct {
	Objects []JSONObject `json:"objects"`
}

func ReplaceIndexInIndexPattern(index string, content common.MapStr) (err error) {
	if index == "" {
		return nil
	}

	list, ok := content["objects"]
	if !ok {
		return errors.New("empty index pattern")
	}

	updateObject := func(obj common.MapStr) {
		// This uses Put instead of DeepUpdate to avoid modifying types for
		// inner objects. (DeepUpdate will replace maps with MapStr).
		obj.Put("id", index)
		// Only overwrite title if it exists.
		if _, err := obj.GetValue("attributes.title"); err == nil {
			obj.Put("attributes.title", index)
		}
	}

	switch v := list.(type) {
	case []interface{}:
		for _, objIf := range v {
			switch obj := objIf.(type) {
			case common.MapStr:
				updateObject(obj)
			case map[string]interface{}:
				updateObject(obj)
			default:
				return errors.Errorf("index pattern object has unexpected type %T", v)
			}
		}
	case []map[string]interface{}:
		for _, obj := range v {
			updateObject(obj)
		}
	case []common.MapStr:
		for _, obj := range v {
			updateObject(obj)
		}
	default:
		return errors.Errorf("index pattern objects have unexpected type %T", v)
	}
	return nil
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
	if visStateJSON, ok := kibanaSavedObject["visState"].(string); ok {
		visStateJSON = ReplaceIndexInVisState(index, visStateJSON)
		kibanaSavedObject["visState"] = visStateJSON
	}

	return kibanaSavedObject
}

// ReplaceIndexInVisState replaces index appearing in visState params objects
func ReplaceIndexInVisState(index string, visStateJSON string) string {

	var visState map[string]interface{}
	err := json.Unmarshal([]byte(visStateJSON), &visState)
	if err != nil {
		logp.Err("Fail to unmarshal visState: %v", err)
		return visStateJSON
	}

	params, ok := visState["params"].(map[string]interface{})
	if !ok {
		return visStateJSON
	}

	// Don't set it if it was not set before
	if pattern, ok := params["index_pattern"].(string); !ok || len(pattern) == 0 {
		return visStateJSON
	}

	params["index_pattern"] = index

	d, err := json.Marshal(visState)
	if err != nil {
		logp.Err("Fail to marshal visState: %v", err)
		return visStateJSON
	}

	return string(d)
}

// ReplaceIndexInDashboardObject replaces references to the index pattern in dashboard objects
func ReplaceIndexInDashboardObject(index string, content []byte) []byte {
	if index == "" {
		return content
	}

	if len(bytes.TrimSpace(content)) == 0 {
		return content
	}

	objectMap := make(map[string]interface{}, 0)
	err := json.Unmarshal(content, &objectMap)
	if err != nil {
		logp.Err("Failed to convert bytes to map[string]interface: %+v", err)
		return content
	}

	attributes, ok := objectMap["attributes"].(map[string]interface{})
	if !ok {
		logp.Err("Object does not have attributes key")
		return content
	}

	if kibanaSavedObject, ok := attributes["kibanaSavedObjectMeta"].(map[string]interface{}); ok {
		attributes["kibanaSavedObjectMeta"] = ReplaceIndexInSavedObject(index, kibanaSavedObject)
	}

	if visState, ok := attributes["visState"].(string); ok {
		attributes["visState"] = ReplaceIndexInVisState(index, visState)
	}

	b, err := json.Marshal(objectMap)
	if err != nil {
		logp.Err("Error marshaling modified dashboard: %+v", err)
		return content
	}

	return b
}

func EncodeJSONObjects(content []byte) []byte {
	logger := logp.NewLogger("dashboards")

	if len(bytes.TrimSpace(content)) == 0 {
		return content
	}

	objectMap := make(map[string]interface{}, 0)
	err := json.Unmarshal(content, &objectMap)
	if err != nil {
		logger.Errorf("Failed to convert bytes to map[string]interface: %+v", err)
		return content
	}

	attributes, ok := objectMap["attributes"].(map[string]interface{})
	if !ok {
		logger.Errorf("Object does not have attributes key")
		return content
	}

	if kibanaSavedObject, ok := attributes["kibanaSavedObjectMeta"].(map[string]interface{}); ok {
		if searchSourceJSON, ok := kibanaSavedObject["searchSourceJSON"].(map[string]interface{}); ok {
			b, err := json.Marshal(searchSourceJSON)
			if err != nil {
				return content
			}
			kibanaSavedObject["searchSourceJSON"] = string(b)
		}
	}

	fieldsToStr := []string{"visState", "uiStateJSON", "optionsJSON"}
	for _, field := range fieldsToStr {
		if rootField, ok := attributes[field].(map[string]interface{}); ok {
			b, err := json.Marshal(rootField)
			if err != nil {
				return content
			}
			attributes[field] = string(b)
		}
	}

	if panelsJSON, ok := attributes["panelsJSON"].([]interface{}); ok {
		b, err := json.Marshal(panelsJSON)
		if err != nil {
			return content
		}
		attributes["panelsJSON"] = string(b)

	}

	b, err := json.Marshal(objectMap)
	if err != nil {
		logger.Error("Error marshaling modified dashboard: %+v", err)
		return content
	}

	return b

}

// ReplaceStringInDashboard replaces a string field in a dashboard
func ReplaceStringInDashboard(old, new string, content []byte) []byte {
	return bytes.Replace(content, []byte(old), []byte(new), -1)
}
