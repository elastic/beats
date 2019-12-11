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
	"github.com/elastic/beats/libbeat/mapping"
)

// Processor struct to process fields to template
type Processor struct {
	EsVersion common.Version
	Migration bool
}

var (
	defaultScalingFactor = 1000
	defaultIgnoreAbove   = 1024
)

const scalingFactorKey = "scalingFactor"

type fieldState struct {
	DefaultField bool
	Path         string
}

// Process recursively processes the given fields and writes the template in the given output
func (p *Processor) Process(fields mapping.Fields, state *fieldState, output common.MapStr) error {
	if state == nil {
		// Set the defaults.
		state = &fieldState{DefaultField: true}
	}

	for _, field := range fields {
		if field.Name == "" {
			continue
		}

		field.Path = state.Path
		if field.DefaultField == nil {
			field.DefaultField = &state.DefaultField
		}
		var indexMapping common.MapStr

		switch field.Type {
		case "ip":
			indexMapping = p.ip(&field)
		case "scaled_float":
			indexMapping = p.scaledFloat(&field)
		case "half_float":
			indexMapping = p.halfFloat(&field)
		case "integer":
			indexMapping = p.integer(&field)
		case "text":
			indexMapping = p.text(&field)
		case "", "keyword":
			indexMapping = p.keyword(&field)
		case "object":
			indexMapping = p.object(&field)
		case "array":
			indexMapping = p.array(&field)
		case "alias":
			indexMapping = p.alias(&field)
		case "group":
			indexMapping = common.MapStr{}
			if field.Dynamic.Value != nil {
				indexMapping["dynamic"] = field.Dynamic.Value
			}

			// Combine properties with previous field definitions (if any)
			properties := common.MapStr{}
			key := mapping.GenerateKey(field.Name) + ".properties"
			currentProperties, err := output.GetValue(key)
			if err == nil {
				var ok bool
				properties, ok = currentProperties.(common.MapStr)
				if !ok {
					// This should never happen
					return errors.New(key + " is expected to be a MapStr")
				}
			}

			groupState := &fieldState{Path: field.Name, DefaultField: *field.DefaultField}
			if state.Path != "" {
				groupState.Path = state.Path + "." + field.Name
			}
			if err := p.Process(field.Fields, groupState, properties); err != nil {
				return err
			}
			indexMapping["properties"] = properties
		default:
			indexMapping = p.other(&field)
		}

		if *field.DefaultField {
			switch field.Type {
			case "", "keyword", "text":
				addToDefaultFields(&field)
			}
		}

		if len(indexMapping) > 0 {
			output.Put(mapping.GenerateKey(field.Name), indexMapping)
		}
	}
	return nil
}

func addToDefaultFields(f *mapping.Field) {
	fullName := f.Name
	if f.Path != "" {
		fullName = f.Path + "." + f.Name
	}

	if f.Index == nil || (f.Index != nil && *f.Index) {
		defaultFields = append(defaultFields, fullName)
	}
}

func (p *Processor) other(f *mapping.Field) common.MapStr {
	property := getDefaultProperties(f)
	if f.Type != "" {
		property["type"] = f.Type
	}

	return property
}

func (p *Processor) integer(f *mapping.Field) common.MapStr {
	property := getDefaultProperties(f)
	property["type"] = "long"
	return property
}

func (p *Processor) scaledFloat(f *mapping.Field, params ...common.MapStr) common.MapStr {
	property := getDefaultProperties(f)
	property["type"] = "scaled_float"

	if p.EsVersion.IsMajor(2) {
		property["type"] = "float"
		return property
	}

	// Set scaling factor
	scalingFactor := defaultScalingFactor
	if f.ScalingFactor != 0 && len(f.ObjectTypeParams) == 0 {
		scalingFactor = f.ScalingFactor
	}

	if len(params) > 0 {
		if s, ok := params[0][scalingFactorKey].(int); ok && s != 0 {
			scalingFactor = s
		}
	}

	property["scaling_factor"] = scalingFactor
	return property
}

func (p *Processor) halfFloat(f *mapping.Field) common.MapStr {
	property := getDefaultProperties(f)
	property["type"] = "half_float"

	if p.EsVersion.IsMajor(2) {
		property["type"] = "float"
	}
	return property
}

func (p *Processor) ip(f *mapping.Field) common.MapStr {
	property := getDefaultProperties(f)

	property["type"] = "ip"

	if p.EsVersion.IsMajor(2) {
		property["type"] = "string"
		property["ignore_above"] = 1024
		property["index"] = "not_analyzed"
	}
	return property
}

func (p *Processor) keyword(f *mapping.Field) common.MapStr {
	property := getDefaultProperties(f)

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
		p.Process(f.MultiFields, nil, fields)
		property["fields"] = fields
	}

	return property
}

func (p *Processor) text(f *mapping.Field) common.MapStr {
	properties := getDefaultProperties(f)

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
		p.Process(f.MultiFields, nil, fields)
		properties["fields"] = fields
	}

	return properties
}

func (p *Processor) array(f *mapping.Field) common.MapStr {
	properties := getDefaultProperties(f)
	if f.ObjectType != "" {
		properties["type"] = f.ObjectType
	}
	return properties
}

func (p *Processor) alias(f *mapping.Field) common.MapStr {
	// Aliases were introduced in Elasticsearch 6.4, ignore if unsupported
	if p.EsVersion.LessThan(common.MustNewVersion("6.4.0")) {
		return nil
	}

	// In case migration is disabled and it's a migration alias, field is not created
	if !p.Migration && f.MigrationAlias {
		return nil
	}
	properties := getDefaultProperties(f)
	properties["type"] = "alias"
	properties["path"] = f.AliasPath

	return properties
}

func (p *Processor) object(f *mapping.Field) common.MapStr {
	matchType := func(onlyType string, mt string) string {
		if mt != "" {
			return mt
		}
		return onlyType
	}

	var otParams []mapping.ObjectTypeCfg
	if len(f.ObjectTypeParams) != 0 {
		otParams = f.ObjectTypeParams
	} else {
		otParams = []mapping.ObjectTypeCfg{mapping.ObjectTypeCfg{
			ObjectType: f.ObjectType, ObjectTypeMappingType: f.ObjectTypeMappingType, ScalingFactor: f.ScalingFactor}}
	}

	for _, otp := range otParams {
		dynProperties := getDefaultProperties(f)

		switch otp.ObjectType {
		case "scaled_float":
			dynProperties = p.scaledFloat(f, common.MapStr{scalingFactorKey: otp.ScalingFactor})
			addDynamicTemplate(f, dynProperties, matchType("*", otp.ObjectTypeMappingType))
		case "text":
			dynProperties["type"] = "text"

			if p.EsVersion.IsMajor(2) {
				dynProperties["type"] = "string"
				dynProperties["index"] = "analyzed"
			}
			addDynamicTemplate(f, dynProperties, matchType("string", otp.ObjectTypeMappingType))
		case "keyword":
			dynProperties["type"] = otp.ObjectType
			addDynamicTemplate(f, dynProperties, matchType("string", otp.ObjectTypeMappingType))
		case "byte", "double", "float", "long", "short", "boolean":
			dynProperties["type"] = otp.ObjectType
			addDynamicTemplate(f, dynProperties, matchType(otp.ObjectType, otp.ObjectTypeMappingType))
		}
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

func addDynamicTemplate(f *mapping.Field, properties common.MapStr, matchType string) {
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

func getDefaultProperties(f *mapping.Field) common.MapStr {
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
