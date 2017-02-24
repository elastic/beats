package template

import (
	"strings"

	"github.com/elastic/beats/libbeat/common"
)

var (
	defaultScalingFactor = 1000
)

type Field struct {
	Name          string `config:"name"`
	Type          string `config:"type"`
	Description   string `config:"description"`
	Format        string `config:"format"`
	ScalingFactor int    `config:"scaling_factor"`
	Fields        Fields `config:"fields"`
	ObjectType    string `config:"object_type"`

	path      string
	esVersion Version
}

// This includes all entries without special handling for different versions.
// Currently this is:
// long, geo_point, date, short, byte, float, double, boolean
func (f *Field) other() common.MapStr {
	property := getDefaultProperties()
	property["type"] = f.Type
	return property
}

func (f *Field) integer() common.MapStr {
	property := getDefaultProperties()
	property["type"] = "long"
	return property
}

func (f *Field) scaledFloat() common.MapStr {
	property := getDefaultProperties()
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
	property := getDefaultProperties()
	property["type"] = "half_float"

	if f.esVersion.IsMajor(2) {
		property["type"] = "float"
	}
	return property
}

func (f *Field) ip() common.MapStr {
	property := getDefaultProperties()

	property["type"] = "ip"

	if f.esVersion.IsMajor(2) {
		property["type"] = "string"
		property["ignore_above"] = 1024
		property["index"] = "not_analyzed"
	}
	return property
}

func (f *Field) keyword() common.MapStr {
	property := getDefaultProperties()

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
	property := getDefaultProperties()

	property["type"] = "text"

	if f.esVersion.IsMajor(2) {
		property["type"] = "string"
		property["index"] = "analyzed"
		property["norms"] = common.MapStr{
			"enabled": false,
		}
	} else {
		property["norms"] = false
	}
	return property
}

func (f *Field) array() common.MapStr {
	return common.MapStr{
		"properties": common.MapStr{},
	}
}

func (f *Field) object() common.MapStr {

	if f.ObjectType == "text" {
		properties := getDefaultProperties()
		properties["type"] = "text"

		if f.esVersion.IsMajor(2) {
			properties["type"] = "string"
			properties["index"] = "analyzed"
		}
		f.addDynamicTemplate(properties, "string")
	}

	if f.ObjectType == "long" {
		properties := getDefaultProperties()
		properties["type"] = "long"
		f.addDynamicTemplate(properties, "long")
	}
	return common.MapStr{
		"properties": common.MapStr{},
	}
}

func (f *Field) addDynamicTemplate(properties common.MapStr, matchType string) {

	template := common.MapStr{
		// Set the path of the field as name
		f.path + "." + f.Name: common.MapStr{
			"mapping":            properties,
			"match_mapping_type": matchType,
			"path_match":         f.path + "." + f.Name + ".*",
		},
	}

	dynamicTemplates = append(dynamicTemplates, template)
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

func getDefaultProperties() common.MapStr {
	// Currently no defaults exist
	return common.MapStr{}
}
