package common

import (
	"fmt"
	"strings"

	"github.com/elastic/go-ucfg/yaml"
)

//This reflects allowed attributes for field definitions in the fields.yml.
//No logic is put into this data structure.
//The purpose is to enable using different kinds of transformation, on top of the same data structure.
//Current transformation:
//  -ElasticSearch Template
//  -Kibana Index Pattern

type Fields []Field

type Field struct {
	Name                  string      `config:"name"`
	Type                  string      `config:"type"`
	Description           string      `config:"description"`
	Format                string      `config:"format"`
	ScalingFactor         int         `config:"scaling_factor"`
	Fields                Fields      `config:"fields"`
	MultiFields           Fields      `config:"multi_fields"`
	ObjectType            string      `config:"object_type"`
	ObjectTypeMappingType string      `config:"object_type_mapping_type"`
	Enabled               *bool       `config:"enabled"`
	Analyzer              string      `config:"analyzer"`
	SearchAnalyzer        string      `config:"search_analyzer"`
	Norms                 bool        `config:"norms"`
	Dynamic               DynamicType `config:"dynamic"`
	Index                 *bool       `config:"index"`
	DocValues             *bool       `config:"doc_values"`
	CopyTo                string      `config:"copy_to"`

	// Kibana specific
	Analyzed     *bool  `config:"analyzed"`
	Count        int    `config:"count"`
	Searchable   *bool  `config:"searchable"`
	Aggregatable *bool  `config:"aggregatable"`
	Script       string `config:"script"`
	// Kibana params
	Pattern              string              `config:"pattern"`
	InputFormat          string              `config:"input_format"`
	OutputFormat         string              `config:"output_format"`
	OutputPrecision      *int                `config:"output_precision"`
	LabelTemplate        string              `config:"label_template"`
	UrlTemplate          []VersionizedString `config:"url_template"`
	OpenLinkInCurrentTab *bool               `config:"open_link_in_current_tab"`

	Path string
}

type VersionizedString struct {
	MinVersion string `config:"min_version"`
	Value      string `config:"value"`
}

type DynamicType struct{ Value interface{} }

func (d *DynamicType) Unpack(s string) error {
	switch s {
	case "true":
		d.Value = true
	case "false":
		d.Value = false
	case "strict":
		d.Value = s
	default:
		return fmt.Errorf("'%v' is invalid dynamic setting", s)
	}
	return nil
}

func LoadFieldsYaml(path string) (Fields, error) {
	keys := []Field{}

	cfg, err := yaml.NewConfigWithFile(path)
	if err != nil {
		return nil, err
	}
	cfg.Unpack(&keys)

	fields := Fields{}

	for _, key := range keys {
		fields = append(fields, key.Fields...)
	}
	return fields, nil
}

// HasKey checks if inside fields the given key exists
// The key can be in the form of a.b.c and it will check if the nested field exist
// In case the key is `a` and there is a value `a.b` false is return as it only
// returns true if it's a leave node
func (f Fields) HasKey(key string) bool {
	keys := strings.Split(key, ".")
	return f.hasKey(keys)
}

// HasNode checks if inside fields the given node exists
// In contrast to HasKey it not only compares the leaf nodes but
// every single key it traverses.
func (f Fields) HasNode(key string) bool {
	keys := strings.Split(key, ".")
	return f.hasNode(keys)
}

func (f Fields) hasNode(keys []string) bool {

	// Nothing to compare, so does not contain it
	if len(keys) == 0 {
		return false
	}

	key := keys[0]
	keys = keys[1:]

	for _, field := range f {

		if field.Name == key {

			//// It's the last key to compare
			if len(keys) == 0 {
				return true
			}

			// It's the last field to compare
			if len(field.Fields) == 0 {
				return true
			}

			return field.Fields.hasNode(keys)
		}
	}
	return false
}

// Recursively generates the correct key based on the dots
// The mapping requires "properties" between each layer. This is added here.
func GenerateKey(key string) string {
	if strings.Contains(key, ".") {
		keys := strings.SplitN(key, ".", 2)
		key = keys[0] + ".properties." + GenerateKey(keys[1])
	}
	return key
}

func (f Fields) hasKey(keys []string) bool {
	// Nothing to compare anymore
	if len(keys) == 0 {
		return false
	}

	key := keys[0]
	keys = keys[1:]

	for _, field := range f {
		if field.Name == key {

			if len(field.Fields) > 0 {
				return field.Fields.hasKey(keys)
			}
			// Last entry in the tree but still more keys
			if len(keys) > 0 {
				return false
			}

			return true
		}
	}
	return false
}

// GetKeys returns a flat list of keys this Fields contains
func (f Fields) GetKeys() []string {
	return f.getKeys("")
}

func (f Fields) getKeys(namespace string) []string {

	var keys []string

	for _, field := range f {
		fieldName := namespace + "." + field.Name
		if namespace == "" {
			fieldName = field.Name
		}
		if len(field.Fields) == 0 {
			keys = append(keys, fieldName)
		} else {
			keys = append(keys, field.Fields.getKeys(fieldName)...)
		}
	}

	return keys
}
