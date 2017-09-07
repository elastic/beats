package template

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

var (
	defaultScalingFactor = 1000
)

type Field struct {
	Name           string      `config:"name"`
	Type           string      `config:"type"`
	Description    string      `config:"description"`
	Format         string      `config:"format"`
	ScalingFactor  int         `config:"scaling_factor"`
	Fields         Fields      `config:"fields"`
	MultiFields    Fields      `config:"multi_fields"`
	ObjectType     string      `config:"object_type"`
	Enabled        *bool       `config:"enabled"`
	Analyzer       string      `config:"analyzer"`
	SearchAnalyzer string      `config:"search_analyzer"`
	Norms          bool        `config:"norms"`
	Dynamic        dynamicType `config:"dynamic"`
	Index          *bool       `config:"index"`
	DocValues      *bool       `config:"doc_values"`

	path      string
	esVersion common.Version
}

// This includes all entries without special handling for different versions.
// Currently this is:
// long, geo_point, date, short, byte, float, double, boolean
func (f *Field) other() common.MapStr {
	property := f.getDefaultProperties()
	if f.Type != "" {
		property["type"] = f.Type
	}

	return property
}

func (f *Field) integer() common.MapStr {
	property := f.getDefaultProperties()
	property["type"] = "long"
	return property
}

func (f *Field) scaledFloat() common.MapStr {
	property := f.getDefaultProperties()
	property["type"] = "scaled_float"

	if f.esVersion.IsMajor(2) {
		property["type"] = "float"
	} else {
		// Set default scaling factor
		if f.ScalingFactor == 0 {
			f.ScalingFactor = defaultScalingFactor
		}
		property["scaling_factor"] = f.ScalingFactor
	}
	return property
}

func (f *Field) halfFloat() common.MapStr {
	property := f.getDefaultProperties()
	property["type"] = "half_float"

	if f.esVersion.IsMajor(2) {
		property["type"] = "float"
	}
	return property
}

func (f *Field) ip() common.MapStr {
	property := f.getDefaultProperties()

	property["type"] = "ip"

	if f.esVersion.IsMajor(2) {
		property["type"] = "string"
		property["ignore_above"] = 1024
		property["index"] = "not_analyzed"
	}
	return property
}

func (f *Field) keyword() common.MapStr {
	property := f.getDefaultProperties()

	property["type"] = "keyword"
	property["ignore_above"] = 1024

	if f.esVersion.IsMajor(2) {
		property["type"] = "string"
		property["ignore_above"] = 1024
		property["index"] = "not_analyzed"
	}
	return property
}

func (f *Field) text() common.MapStr {
	properties := f.getDefaultProperties()

	properties["type"] = "text"

	if f.esVersion.IsMajor(2) {
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
		f.MultiFields.process("", f.esVersion, fields)
		properties["fields"] = fields
	}

	return properties
}

func (f *Field) array() common.MapStr {
	properties := f.getDefaultProperties()
	if f.ObjectType != "" {
		properties["type"] = f.ObjectType
	}
	return properties
}

func (f *Field) object() common.MapStr {
	dynProperties := f.getDefaultProperties()

	switch f.ObjectType {
	case "text":
		dynProperties["type"] = "text"

		if f.esVersion.IsMajor(2) {
			dynProperties["type"] = "string"
			dynProperties["index"] = "analyzed"
		}
		f.addDynamicTemplate(dynProperties, "string")
	case "long":
		dynProperties["type"] = f.ObjectType
		f.addDynamicTemplate(dynProperties, "long")
	case "keyword":
		dynProperties["type"] = f.ObjectType
		f.addDynamicTemplate(dynProperties, "string")
	}

	properties := f.getDefaultProperties()
	properties["type"] = "object"
	if f.Enabled != nil {
		properties["enabled"] = *f.Enabled
	}

	if f.Dynamic.value != nil {
		properties["dynamic"] = f.Dynamic.value
	}

	return properties
}

func (f *Field) addDynamicTemplate(properties common.MapStr, matchType string) {
	path := ""
	if len(f.path) > 0 {
		path = f.path + "."
	}
	template := common.MapStr{
		// Set the path of the field as name
		path + f.Name: common.MapStr{
			"mapping":            properties,
			"match_mapping_type": matchType,
			"path_match":         path + f.Name + ".*",
		},
	}

	dynamicTemplates = append(dynamicTemplates, template)
}

func (f *Field) getDefaultProperties() common.MapStr {
	// Currently no defaults exist
	properties := common.MapStr{}

	if f.Index != nil {
		properties["index"] = *f.Index
	}

	if f.DocValues != nil {
		properties["doc_values"] = *f.DocValues
	}
	return properties
}

// Recursively generates the correct key based on the dots
// The mapping requires "properties" between each layer. This is added here.
func generateKey(key string) string {
	if strings.Contains(key, ".") {
		keys := strings.SplitN(key, ".", 2)
		key = keys[0] + ".properties." + generateKey(keys[1])
	}
	return key
}

type dynamicType struct{ value interface{} }

func (d *dynamicType) Unpack(s string) error {
	switch s {
	case "true":
		d.value = true
	case "false":
		d.value = false
	case "strict":
		d.value = s
	default:
		return fmt.Errorf("'%v' is invalid dynamic setting", s)
	}
	return nil
}
