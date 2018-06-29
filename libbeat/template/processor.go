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

package template

import (
	"errors"
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

// Processor struct to process fields to template
type Processor struct {
	EsVersion common.Version
}

var (
	defaultScalingFactor = 1000
	defaultIgnoreAbove   = 1024
)

// Process recursively processes the given fields and writes the template in the given output
func (p *Processor) Process(fields common.Fields, path string, output common.MapStr) error {
	for _, field := range fields {

		if field.Name == "" {
			continue
		}

		field.Path = path
		var mapping common.MapStr

		switch field.Type {
		case "ip":
			mapping = p.ip(&field)
		case "scaled_float":
			mapping = p.scaledFloat(&field)
		case "half_float":
			mapping = p.halfFloat(&field)
		case "integer":
			mapping = p.integer(&field)
		case "text":
			mapping = p.text(&field)
		case "", "keyword":
			mapping = p.keyword(&field)
		case "object":
			mapping = p.object(&field)
		case "array":
			mapping = p.array(&field)
		case "group":
			var newPath string
			if path == "" {
				newPath = field.Name
			} else {
				newPath = path + "." + field.Name
			}
			mapping = common.MapStr{}
			if field.Dynamic.Value != nil {
				mapping["dynamic"] = field.Dynamic.Value
			}

			// Combine properties with previous field definitions (if any)
			properties := common.MapStr{}
			key := common.GenerateKey(field.Name) + ".properties"
			currentProperties, err := output.GetValue(key)
			if err == nil {
				var ok bool
				properties, ok = currentProperties.(common.MapStr)
				if !ok {
					// This should never happen
					return errors.New(key + " is expected to be a MapStr")
				}
			}

			if err := p.Process(field.Fields, newPath, properties); err != nil {
				return err
			}
			mapping["properties"] = properties
		default:
			mapping = p.other(&field)
		}

		if len(mapping) > 0 {
			output.Put(common.GenerateKey(field.Name), mapping)
		}
	}
	return nil
}

func (p *Processor) other(f *common.Field) common.MapStr {
	property := getDefaultProperties(f)
	if f.Type != "" {
		property["type"] = f.Type
	}

	return property
}

func (p *Processor) integer(f *common.Field) common.MapStr {
	property := getDefaultProperties(f)
	property["type"] = "long"
	return property
}

func (p *Processor) scaledFloat(f *common.Field) common.MapStr {
	property := getDefaultProperties(f)
	property["type"] = "scaled_float"

	if p.EsVersion.IsMajor(2) {
		property["type"] = "float"
	} else {
		scalingFactor := f.ScalingFactor
		// Set default scaling factor
		if scalingFactor == 0 {
			scalingFactor = defaultScalingFactor
		}
		property["scaling_factor"] = scalingFactor
	}
	return property
}

func (p *Processor) halfFloat(f *common.Field) common.MapStr {
	property := getDefaultProperties(f)
	property["type"] = "half_float"

	if p.EsVersion.IsMajor(2) {
		property["type"] = "float"
	}
	return property
}

func (p *Processor) ip(f *common.Field) common.MapStr {
	property := getDefaultProperties(f)

	property["type"] = "ip"

	if p.EsVersion.IsMajor(2) {
		property["type"] = "string"
		property["ignore_above"] = 1024
		property["index"] = "not_analyzed"
	}
	return property
}

func (p *Processor) keyword(f *common.Field) common.MapStr {
	property := getDefaultProperties(f)

	fullName := f.Name
	if f.Path != "" {
		fullName = f.Path + "." + f.Name
	}

	defaultFields = append(defaultFields, fullName)

	property["type"] = "keyword"

	switch f.IgnoreAbove {
	case 0: // Use libbeat default
		property["ignore_above"] = defaultIgnoreAbove
	case -1: // Use ES default
	default: // Use user value
		property["ignore_above"] = f.IgnoreAbove
	}

	if p.EsVersion.IsMajor(2) {
		property["type"] = "string"
		property["index"] = "not_analyzed"
	}

	if len(f.MultiFields) > 0 {
		fields := common.MapStr{}
		p.Process(f.MultiFields, "", fields)
		property["fields"] = fields
	}

	return property
}

func (p *Processor) text(f *common.Field) common.MapStr {
	properties := getDefaultProperties(f)

	fullName := f.Name
	if f.Path != "" {
		fullName = f.Path + "." + f.Name
	}

	defaultFields = append(defaultFields, fullName)

	properties["type"] = "text"

	if p.EsVersion.IsMajor(2) {
		properties["type"] = "string"
		properties["index"] = "analyzed"
		if !f.Norms {
			properties["norms"] = common.MapStr{
				"enabled": false,
			}
		}
	} else {
		if !f.Norms {
			properties["norms"] = false
		}
	}

	if f.Analyzer != "" {
		properties["analyzer"] = f.Analyzer
	}

	if f.SearchAnalyzer != "" {
		properties["search_analyzer"] = f.SearchAnalyzer
	}

	if len(f.MultiFields) > 0 {
		fields := common.MapStr{}
		p.Process(f.MultiFields, "", fields)
		properties["fields"] = fields
	}

	return properties
}

func (p *Processor) array(f *common.Field) common.MapStr {
	properties := getDefaultProperties(f)
	if f.ObjectType != "" {
		properties["type"] = f.ObjectType
	}
	return properties
}

func (p *Processor) object(f *common.Field) common.MapStr {
	dynProperties := getDefaultProperties(f)

	matchType := func(onlyType string) string {
		if f.ObjectTypeMappingType != "" {
			return f.ObjectTypeMappingType
		}
		return onlyType
	}

	switch f.ObjectType {
	case "scaled_float":
		dynProperties = p.scaledFloat(f)
		addDynamicTemplate(f, dynProperties, matchType("*"))
	case "text":
		dynProperties["type"] = "text"

		if p.EsVersion.IsMajor(2) {
			dynProperties["type"] = "string"
			dynProperties["index"] = "analyzed"
		}
		addDynamicTemplate(f, dynProperties, matchType("string"))
	case "keyword":
		dynProperties["type"] = f.ObjectType
		addDynamicTemplate(f, dynProperties, matchType("string"))
	case "byte", "double", "float", "long", "short":
		dynProperties["type"] = f.ObjectType
		addDynamicTemplate(f, dynProperties, matchType(f.ObjectType))
	}

	properties := getDefaultProperties(f)
	properties["type"] = "object"
	if f.Enabled != nil {
		properties["enabled"] = *f.Enabled
	}

	if f.Dynamic.Value != nil {
		properties["dynamic"] = f.Dynamic.Value
	}

	return properties
}

func addDynamicTemplate(f *common.Field, properties common.MapStr, matchType string) {
	path := ""
	if len(f.Path) > 0 {
		path = f.Path + "."
	}
	pathMatch := path + f.Name
	if !strings.ContainsRune(pathMatch, '*') {
		pathMatch += ".*"
	}
	template := common.MapStr{
		// Set the path of the field as name
		path + f.Name: common.MapStr{
			"mapping":            properties,
			"match_mapping_type": matchType,
			"path_match":         pathMatch,
		},
	}

	dynamicTemplates = append(dynamicTemplates, template)
}

func getDefaultProperties(f *common.Field) common.MapStr {
	// Currently no defaults exist
	properties := common.MapStr{}

	if f.Index != nil {
		properties["index"] = *f.Index
	}

	if f.DocValues != nil {
		properties["doc_values"] = *f.DocValues
	}

	if f.CopyTo != "" {
		properties["copy_to"] = f.CopyTo
	}
	return properties
}
