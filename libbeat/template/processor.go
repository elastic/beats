package template

import (
	"errors"
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

type Processor struct {
	EsVersion common.Version
}

var (
	defaultScalingFactor = 1000
)

// This includes all entries without special handling for different versions.
// Currently this is:
// long, geo_point, date, short, byte, float, double, boolean
func (p *Processor) process(fields common.Fields, path string, output common.MapStr) error {
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

			if err := p.process(field.Fields, newPath, properties); err != nil {
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

	property["type"] = "keyword"
	property["ignore_above"] = 1024

	if p.EsVersion.IsMajor(2) {
		property["type"] = "string"
		property["ignore_above"] = 1024
		property["index"] = "not_analyzed"
	}
	return property
}

func (p *Processor) text(f *common.Field) common.MapStr {
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
		p.process(f.MultiFields, "", fields)
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
	case "long":
		dynProperties["type"] = f.ObjectType
		addDynamicTemplate(f, dynProperties, matchType("long"))
	case "keyword":
		dynProperties["type"] = f.ObjectType
		addDynamicTemplate(f, dynProperties, matchType("string"))
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
