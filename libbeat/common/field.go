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
//  -Kibana Index Pattern
//  -ElasticSearch Template: still uses its own Fields definition

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
	Dynamic        DynamicType `config:"dynamic"`
	Index          *bool       `config:"index"`
	DocValues      *bool       `config:"doc_values"`

	Path string
}

type Fields []Field

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

// Recursively generates the correct key based on the dots
// The mapping requires "properties" between each layer. This is added here.
func GenerateKey(key string) string {
	if strings.Contains(key, ".") {
		keys := strings.SplitN(key, ".", 2)
		key = keys[0] + ".properties." + GenerateKey(keys[1])
	}
	return key
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
