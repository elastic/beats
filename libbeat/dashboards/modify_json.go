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
	"regexp"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
func ReplaceIndexInIndexPattern(index string, content mapstr.M) (err error) {
	if index == "" {
		return nil
	}

	// This uses Put instead of DeepUpdate to avoid modifying types for
	// inner objects. (DeepUpdate will replace maps with MapStr).
	content.Put("id", index)
	// Only overwrite title if it exists.
	if _, err := content.GetValue("attributes.title"); err == nil {
		content.Put("attributes.title", index)
	}

	return nil
}

func replaceIndexInSearchObject(index string, savedObject string) (string, error) {

	var record mapstr.M
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

var timeLionIdxRegexp = regexp.MustCompile(`index=\".*beat-\*\"`)

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

	if s, ok := params["series"].([]interface{}); ok {
		for i, ser := range s {
			if series, ok := ser.(map[string]interface{}); ok {
				if _, ok := series["series_index_pattern"]; !ok {
					continue
				}
				series["series_index_pattern"] = index
				s[i] = series
			}
		}
		params["series"] = s
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

	if expr, ok := params["expression"].(string); ok {
		params["expression"] = timeLionIdxRegexp.ReplaceAllString(expr, `index="`+index+`"`)
	}

	visState["params"] = replaceIndexInParamControls(logger, index, params)

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

	if layerListJSON, ok := attributes["layerListJSON"].([]interface{}); ok {
		attributes["layerListJSON"] = replaceIndexInLayerListJSON(logger, index, layerListJSON)
	}

	if mapStateJSON, ok := attributes["mapStateJSON"].(map[string]interface{}); ok {
		attributes["mapStateJSON"] = replaceIndexInMapStateJSON(logger, index, mapStateJSON)
	}

	if panelsJSON, ok := attributes["panelsJSON"].([]interface{}); ok {
		attributes["panelsJSON"] = replaceIndexInPanelsJSON(logger, index, panelsJSON)
	}

	objectMap["attributes"] = attributes

	if references, ok := objectMap["references"].([]interface{}); ok {
		objectMap["references"] = replaceIndexInReferences(index, references)
	}

	b, err := json.Marshal(objectMap)
	if err != nil {
		logger.Error("Error marshaling modified dashboard: %+v", err)
		return content
	}

	return b
}

func replaceIndexInLayerListJSON(logger *logp.Logger, index string, layerListJSON []interface{}) []interface{} {
	for i, layerListElem := range layerListJSON {
		elem, ok := layerListElem.(map[string]interface{})
		if !ok {
			continue
		}

		if joins, ok := elem["joins"].([]interface{}); ok {
			for j, join := range joins {
				if pos, ok := join.(map[string]interface{}); ok {
					for key, val := range pos {
						if joinElems, ok := val.(map[string]interface{}); ok {
							if _, ok := joinElems["indexPatternTitle"]; ok {
								joinElems["indexPatternTitle"] = index
								pos[key] = joinElems
							}
						}
					}
					joins[j] = pos
				}
			}
			elem["joins"] = joins
		}
		if descriptor, ok := elem["sourceDescriptor"].(map[string]interface{}); ok {
			if _, ok := descriptor["indexPatternId"]; ok {
				descriptor["indexPatternId"] = index
			}
			elem["sourceDescriptor"] = descriptor
		}

		layerListJSON[i] = elem
	}
	return layerListJSON
}

func replaceIndexInMapStateJSON(logger *logp.Logger, index string, mapState map[string]interface{}) map[string]interface{} {
	if filters, ok := mapState["filters"].([]interface{}); ok {
		for i, f := range filters {
			if filter, ok := f.(map[string]interface{}); ok {
				if meta, ok := filter["meta"].(map[string]interface{}); ok {
					if _, ok := meta["index"]; !ok {
						continue
					}
					meta["index"] = index
					filter["meta"] = meta
				}
				filters[i] = filter
			}
		}
		mapState["filters"] = filters
	}

	return mapState
}

func replaceIndexInPanelsJSON(logger *logp.Logger, index string, panelsJSON []interface{}) []interface{} {
	for i, p := range panelsJSON {
		if panel, ok := p.(map[string]interface{}); ok {
			config, ok := panel["embeddableConfig"].(map[string]interface{})
			if !ok {
				continue
			}
			if configAttr, ok := config["attributes"].(map[string]interface{}); ok {
				if references, ok := configAttr["references"].([]interface{}); ok {
					configAttr["references"] = replaceIndexInReferences(index, references)
				}
				if layerListJSON, ok := configAttr["layerListJSON"].([]interface{}); ok {
					configAttr["layerListJSON"] = replaceIndexInLayerListJSON(logger, index, layerListJSON)
				}
				config["attributes"] = configAttr
			}

			if savedVis, ok := config["savedVis"].(map[string]interface{}); ok {
				if params, ok := savedVis["params"].(map[string]interface{}); ok {
					savedVis["params"] = replaceIndexInParamControls(logger, index, params)
				}
				config["savedVis"] = savedVis
			}

			panel["embeddableConfig"] = config
			panelsJSON[i] = panel
		}
	}
	return panelsJSON
}

func replaceIndexInParamControls(logger *logp.Logger, index string, params map[string]interface{}) map[string]interface{} {
	if controlsList, ok := params["controls"].([]interface{}); ok {
		for i, ctrl := range controlsList {
			if control, ok := ctrl.(map[string]interface{}); ok {
				if _, ok := control["indexPattern"]; ok {
					control["indexPattern"] = index
					controlsList[i] = control
				}
			}
		}
		params["controls"] = controlsList
	}
	return params
}

func replaceIndexInReferences(index string, references []interface{}) []interface{} {
	for i, ref := range references {
		if reference, ok := ref.(map[string]interface{}); ok {
			if refType, ok := reference["type"].(string); ok {
				if refType == "index-pattern" {
					reference["id"] = index
				}
			}
			references[i] = reference
		}
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
	objectMap["attributes"] = convertAttributes(attributes)

	b, err := json.Marshal(objectMap)
	if err != nil {
		logger.Error("Error marshaling modified dashboard: %+v", err)
		return content
	}

	return b

}

func convertAttributes(attributes map[string]interface{}) map[string]interface{} {
	if kibanaSavedObject, ok := attributes["kibanaSavedObjectMeta"].(map[string]interface{}); ok {
		if searchSourceJSON, ok := kibanaSavedObject["searchSourceJSON"].(map[string]interface{}); ok {
			b, err := json.Marshal(searchSourceJSON)
			if err != nil {
				return attributes
			}
			kibanaSavedObject["searchSourceJSON"] = string(b)
		}
	}

	if panelsJSON, ok := attributes["panelsJSON"].([]interface{}); ok {
		for i, panel := range panelsJSON {
			if panelMap, ok := panel.(map[string]interface{}); ok {
				if embeddableConfig, ok := panelMap["embeddableConfig"].(map[string]interface{}); ok {
					if embeddedAttributes, ok := embeddableConfig["attributes"].(map[string]interface{}); ok {
						embeddableConfig["attributes"] = convertAttributes(embeddedAttributes)
						panelMap["embeddableConfig"] = embeddableConfig
						panelsJSON[i] = panelMap
					}
				}
			}
		}
		attributes["panelsJSON"] = panelsJSON
	}

	attributes = convertObjectsToString(attributes)
	return attributes
}

func convertObjectsToString(attributes map[string]interface{}) map[string]interface{} {
	fieldsToStr := []string{
		"layerListJSON",
		"mapStateJSON",
		"optionsJSON",
		"panelsJSON",
		"uiStateJSON",
		"visState",
	}
	for _, field := range fieldsToStr {
		switch rootField := attributes[field].(type) {
		case map[string]interface{}, []interface{}:
			b, err := json.Marshal(rootField)
			if err != nil {
				return attributes
			}
			attributes[field] = string(b)
		default:
			continue
		}
	}

	return attributes
}

// ReplaceStringInDashboard replaces a string field in a dashboard
func ReplaceStringInDashboard(old, new string, content []byte) []byte {
	return bytes.Replace(content, []byte(old), []byte(new), -1)
}
