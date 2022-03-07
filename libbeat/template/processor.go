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
	"fmt"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/mapping"
)

// DefaultField controls the default value for the default_field flag.
const DefaultField = true

var (
	minVersionAlias                   = common.MustNewVersion("6.4.0")
	minVersionFieldMeta               = common.MustNewVersion("7.6.0")
	minVersionHistogram               = common.MustNewVersion("7.6.0")
	minVersionWildcard                = common.MustNewVersion("7.9.0")
	minVersionExplicitDynamicTemplate = common.MustNewVersion("7.13.0")
	minVersionMatchOnlyText           = common.MustNewVersion("7.14.0")
)

// Processor struct to process fields to template
type Processor struct {
	EsVersion       common.Version
	Migration       bool
	ElasticLicensed bool

	// dynamicTemplatesMap records which dynamic templates have been added, to prevent duplicates.
	dynamicTemplatesMap map[dynamicTemplateKey]common.MapStr
	// dynamicTemplates records the dynamic templates in the order they were added.
	dynamicTemplates []common.MapStr
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
		state = &fieldState{DefaultField: DefaultField}
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
		case "match_only_text":
			noMatchOnlyText := p.EsVersion.LessThan(minVersionMatchOnlyText)
			if !p.ElasticLicensed || noMatchOnlyText {
				indexMapping = p.text(&field)
			} else {
				indexMapping = p.matchOnlyText(&field)
			}
		case "wildcard":
			noWildcards := p.EsVersion.LessThan(minVersionWildcard)
			if !p.ElasticLicensed || noWildcards {
				indexMapping = p.keyword(&field)
			} else {
				indexMapping = p.wildcard(&field)
			}
		case "", "keyword":
			indexMapping = p.keyword(&field)
		case "object":
			indexMapping = p.object(&field)
		case "array":
			indexMapping = p.array(&field)
		case "alias":
			indexMapping = p.alias(&field)
		case "histogram":
			indexMapping = p.histogram(&field)
		case "nested":
			mapping, err := p.nested(&field, output)
			if err != nil {
				return err
			}
			indexMapping = mapping
		case "group":
			mapping, err := p.group(&field, output)
			if err != nil {
				return err
			}
			indexMapping = mapping
		default:
			indexMapping = p.other(&field)
		}

		if *field.DefaultField {
			switch field.Type {
			case "", "keyword", "text", "match_only_text", "wildcard":
				addToDefaultFields(&field)
			}
		}

		if len(indexMapping) > 0 {
			if field.DynamicTemplate {
				// Explicit dynamic templates were introduced in
				// Elasticsearch 7.13, ignore if unsupported
				if !p.EsVersion.LessThan(minVersionExplicitDynamicTemplate) {
					p.addDynamicTemplate(field.Name, "", "", indexMapping)
				}
			} else {
				output.Put(mapping.GenerateKey(field.Name), indexMapping)
			}
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
	property := p.getDefaultProperties(f)
	if f.Type != "" {
		property["type"] = f.Type
	}

	return property
}

func (p *Processor) integer(f *mapping.Field) common.MapStr {
	property := p.getDefaultProperties(f)
	property["type"] = "long"
	return property
}

func (p *Processor) scaledFloat(f *mapping.Field, params ...common.MapStr) common.MapStr {
	property := p.getDefaultProperties(f)
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

func (p *Processor) nested(f *mapping.Field, output common.MapStr) (common.MapStr, error) {
	mapping, err := p.group(f, output)
	if err != nil {
		return nil, err
	}
	mapping["type"] = "nested"
	return mapping, nil
}

func (p *Processor) group(f *mapping.Field, output common.MapStr) (common.MapStr, error) {
	indexMapping := common.MapStr{}
	if f.Dynamic.Value != nil {
		indexMapping["dynamic"] = f.Dynamic.Value
	}

	// Combine properties with previous field definitions (if any)
	properties := common.MapStr{}
	key := mapping.GenerateKey(f.Name) + ".properties"
	currentProperties, err := output.GetValue(key)
	if err == nil {
		var ok bool
		properties, ok = currentProperties.(common.MapStr)
		if !ok {
			// This should never happen
			return nil, errors.New(key + " is expected to be a MapStr")
		}
	}

	groupState := &fieldState{Path: f.Name, DefaultField: *f.DefaultField}
	if f.Path != "" {
		groupState.Path = f.Path + "." + f.Name
	}
	if err := p.Process(f.Fields, groupState, properties); err != nil {
		return nil, err
	}
	if len(properties) != 0 {
		indexMapping["properties"] = properties
	}
	return indexMapping, nil
}

func (p *Processor) halfFloat(f *mapping.Field) common.MapStr {
	property := p.getDefaultProperties(f)
	property["type"] = "half_float"

	if p.EsVersion.IsMajor(2) {
		property["type"] = "float"
	}
	return property
}

func (p *Processor) ip(f *mapping.Field) common.MapStr {
	property := p.getDefaultProperties(f)

	property["type"] = "ip"

	if p.EsVersion.IsMajor(2) {
		property["type"] = "string"
		property["ignore_above"] = 1024
		property["index"] = "not_analyzed"
	}
	return property
}

func stateFromField(f *mapping.Field) *fieldState {
	if f == nil {
		return nil
	}
	st := &fieldState{
		DefaultField: DefaultField,
		Path:         f.Name,
	}
	if f.DefaultField != nil {
		st.DefaultField = *f.DefaultField
	}
	if f.Path != "" {
		st.Path = f.Path + "." + f.Name
	}
	return st
}

func (p *Processor) keyword(f *mapping.Field) common.MapStr {
	property := p.getDefaultProperties(f)

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
		p.Process(f.MultiFields, stateFromField(f), fields)
		property["fields"] = fields
	}

	return property
}

func (p *Processor) wildcard(f *mapping.Field) common.MapStr {
	property := p.getDefaultProperties(f)

	property["type"] = "wildcard"

	/* For wildcard fields, unlike keywords, don't force a default ignore_above limit.
	   The default in ES will be used unless an explicit limit is set.
	   This is to take advantage of wildcard type benefits when indexing large strings.
	*/
	if f.IgnoreAbove > 0 {
		property["ignore_above"] = f.IgnoreAbove
	}

	if len(f.MultiFields) > 0 {
		fields := common.MapStr{}
		p.Process(f.MultiFields, stateFromField(f), fields)
		property["fields"] = fields
	}

	return property
}

func (p *Processor) text(f *mapping.Field) common.MapStr {
	properties := p.getDefaultProperties(f)

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
		p.Process(f.MultiFields, stateFromField(f), fields)
		properties["fields"] = fields
	}

	return properties
}

func (p *Processor) matchOnlyText(f *mapping.Field) common.MapStr {
	properties := p.getDefaultProperties(f)

	properties["type"] = "match_only_text"

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
	properties := p.getDefaultProperties(f)
	if f.ObjectType != "" {
		properties["type"] = f.ObjectType
	}
	return properties
}

func (p *Processor) alias(f *mapping.Field) common.MapStr {
	// Aliases were introduced in Elasticsearch 6.4, ignore if unsupported
	if p.EsVersion.LessThan(minVersionAlias) {
		return nil
	}

	// In case migration is disabled and it's a migration alias, field is not created
	if !p.Migration && f.MigrationAlias {
		return nil
	}
	properties := p.getDefaultProperties(f)
	properties["type"] = "alias"
	properties["path"] = f.AliasPath

	return properties
}

func (p *Processor) histogram(f *mapping.Field) common.MapStr {
	// Histograms were introduced in Elasticsearch 7.6, ignore if unsupported
	if p.EsVersion.LessThan(minVersionHistogram) {
		return nil
	}

	properties := p.getDefaultProperties(f)
	properties["type"] = "histogram"

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
		otParams = []mapping.ObjectTypeCfg{{
			ObjectType:            f.ObjectType,
			ObjectTypeMappingType: f.ObjectTypeMappingType,
			ScalingFactor:         f.ScalingFactor,
		}}
	}

	for _, otp := range otParams {
		dynProperties := p.getDefaultProperties(f)
		var matchingType string

		switch otp.ObjectType {
		case "scaled_float":
			dynProperties = p.scaledFloat(f, common.MapStr{scalingFactorKey: otp.ScalingFactor})
			matchingType = matchType("*", otp.ObjectTypeMappingType)
		case "text":
			dynProperties["type"] = "text"

			if p.EsVersion.IsMajor(2) {
				dynProperties["type"] = "string"
				dynProperties["index"] = "analyzed"
			}
			matchingType = matchType("string", otp.ObjectTypeMappingType)
		case "keyword":
			dynProperties["type"] = otp.ObjectType
			matchingType = matchType("string", otp.ObjectTypeMappingType)
		case "byte", "double", "float", "long", "short", "boolean":
			dynProperties["type"] = otp.ObjectType
			matchingType = matchType(otp.ObjectType, otp.ObjectTypeMappingType)
		case "histogram":
			dynProperties["type"] = otp.ObjectType
			matchingType = matchType("*", otp.ObjectTypeMappingType)
		default:
			continue
		}

		path := f.Path
		if len(path) > 0 {
			path += "."
		}
		path += f.Name
		pathMatch := path
		// ensure the `path_match` string ends with a `*`
		if !strings.ContainsRune(path, '*') {
			pathMatch += ".*"
		}
		// When multiple object type parameters are detected for a field,
		// add a unique part to the name of the dynamic template.
		// Duplicated dynamic template names can lead to errors when template
		// inheritance is applied, and will not be supported in future versions
		if len(otParams) > 1 {
			path = fmt.Sprintf("%s_%s", path, matchingType)
		}
		p.addDynamicTemplate(path, pathMatch, matchingType, dynProperties)
	}

	properties := p.getDefaultProperties(f)
	properties["type"] = "object"
	if f.Enabled != nil {
		properties["enabled"] = *f.Enabled
	}

	if f.Dynamic.Value != nil {
		properties["dynamic"] = f.Dynamic.Value
	}

	return properties
}

type dynamicTemplateKey struct {
	name      string
	pathMatch string
	matchType string
}

func (p *Processor) addDynamicTemplate(name, pathMatch, matchType string, properties common.MapStr) {
	key := dynamicTemplateKey{
		name:      name,
		pathMatch: pathMatch,
		matchType: matchType,
	}
	if p.dynamicTemplatesMap == nil {
		p.dynamicTemplatesMap = make(map[dynamicTemplateKey]common.MapStr)
	} else {
		if _, ok := p.dynamicTemplatesMap[key]; ok {
			// Dynamic template already added.
			return
		}
	}
	dynamicTemplateProperties := common.MapStr{
		"mapping": properties,
	}
	if matchType != "" {
		dynamicTemplateProperties["match_mapping_type"] = matchType
	}
	if pathMatch != "" {
		dynamicTemplateProperties["path_match"] = pathMatch
	}
	dynamicTemplate := common.MapStr{
		name: dynamicTemplateProperties,
	}
	p.dynamicTemplatesMap[key] = dynamicTemplate
	p.dynamicTemplates = append(p.dynamicTemplates, dynamicTemplate)
}

func (p *Processor) getDefaultProperties(f *mapping.Field) common.MapStr {
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

	if !p.EsVersion.LessThan(minVersionFieldMeta) {
		if f.MetricType != "" || f.Unit != "" {
			meta := common.MapStr{}
			if f.MetricType != "" {
				meta["metric_type"] = f.MetricType
			}
			if f.Unit != "" {
				meta["unit"] = f.Unit
			}
			properties["meta"] = meta
		}
	}

	return properties
}
