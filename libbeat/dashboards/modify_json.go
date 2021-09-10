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

// JSONObject is an Object with a given JSON attribute
type JSONObject struct {
	Attributes JSONObjectAttribute `json:"attributes"`
}

// ReplaceIndexInIndexPattern replaces an index in a dashboard content body
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

// ReplaceIndexInSavedObject replaces an index in a kibana object
func ReplaceIndexInSavedObject(logger *logp.Logger, index string, kibanaSavedObject map[string]interface{}) map[string]interface{} {

	if searchSourceJSON, ok := kibanaSavedObject["searchSourceJSON"].(string); ok {
		searchSourceJSON, err := replaceIndexInSearchObject(index, searchSourceJSON)
		if err != nil {
			logger.Errorf("Fail to replace searchSourceJSON: %v", err)
			return kibanaSavedObject
		}
		kibanaSavedObject["searchSourceJSON"] = searchSourceJSON
	}
	if visState, ok := kibanaSavedObject["visState"].(map[string]interface{}); ok {
		kibanaSavedObject["visState"] = ReplaceIndexInVisState(logger, index, visState)
	}

	return kibanaSavedObject
}

// ReplaceIndexInVisState replaces index appearing in visState params objects
func ReplaceIndexInVisState(logger *logp.Logger, index string, visState map[string]interface{}) map[string]interface{} {
	params, ok := visState["params"].(map[string]interface{})
	if !ok {
		return visState
	}

	// Don't set it if it was not set before
	if pattern, ok := params["index_pattern"].(string); ok && len(pattern) != 0 {
		params["index_pattern"] = index
	}

	if annotations, ok := params["annotations"].([]interface{}); ok {
		for i, ann := range annotations {
			annotation, ok := ann.(map[string]interface{})
			if !ok {
				continue
			}
			if _, ok = annotation["index_pattern"]; !ok {
				continue
			}
			annotation["index_pattern"] = index
			annotations[i] = annotation
		}
		params["annotations"] = annotations
	}

	visState["params"] = params

	return visState
}

// ReplaceIndexInDashboardObject replaces references to the index pattern in dashboard objects
func ReplaceIndexInDashboardObject(index string, content []byte) []byte {
	logger := logp.NewLogger("dashboards")
	if index == "" {
		return content
	}

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
		attributes["kibanaSavedObjectMeta"] = ReplaceIndexInSavedObject(logger, index, kibanaSavedObject)
	}

	if visState, ok := attributes["visState"].(map[string]interface{}); ok {
		attributes["visState"] = ReplaceIndexInVisState(logger, index, visState)
	}

	if references, ok := objectMap["references"].([]interface{}); ok {
		objectMap["references"] = replaceIndexInReferences(index, references)
	}

	if panelsJSON, ok := objectMap["panelsJSON"].([]interface{}); ok {
		objectMap["panelsJSON"] = replaceIndexInPanelsJSON(logger, index, panelsJSON)
	}

	b, err := json.Marshal(objectMap)
	if err != nil {
		logger.Error("Error marshaling modified dashboard: %+v", err)
		return content
	}

	return b
}

func replaceIndexInPanelsJSON(logger *logp.Logger, index string, panelsJSON []interface{}) []interface{} {
	for i, p := range panelsJSON {
		panel, ok := p.(map[string]interface{})
		if !ok {
			continue
		}

		config, ok := panel["embeddableConfig"].(map[string]interface{})
		if !ok {
			logger.Debugf("panelsJSON.embeddableConfig is not map[string]interface{}")
			return panelsJSON
		}
		configAttr, ok := config["attributes"].(map[string]interface{})
		if !ok {
			logger.Debugf("panelsJSON.embeddableConfig is not map[string]interface{}")
			return panelsJSON
		}
		references, ok := configAttr["references"].([]interface{})
		if !ok {
			logger.Debugf("no panelsJSON.embeddableConfig is not map[string]interface{}")
			return panelsJSON
		}
		configAttr["references"] = replaceIndexInReferences(index, references)
		config["attributes"] = configAttr
		panel["embeddableConfig"] = config
		panelsJSON[i] = panel
	}
	return panelsJSON
}

func replaceIndexInReferences(index string, references []interface{}) []interface{} {
	for i, ref := range references {
		reference, ok := ref.(map[string]interface{})
		if !ok {
			continue
		}

		if refType, ok := reference["type"].(string); ok {
			if refType == "index-pattern" {
				reference["id"] = index
			}
		}
		references[i] = reference
	}
	return references

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
