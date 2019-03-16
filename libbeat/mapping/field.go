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

package mapping

import (
	"fmt"
	"strings"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"

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
	Name           string      `config:"name"`
	Type           string      `config:"type"`
	Description    string      `config:"description"`
	Format         string      `config:"format"`
	Fields         Fields      `config:"fields"`
	MultiFields    Fields      `config:"multi_fields"`
	Enabled        *bool       `config:"enabled"`
	Analyzer       string      `config:"analyzer"`
	SearchAnalyzer string      `config:"search_analyzer"`
	Norms          bool        `config:"norms"`
	Dynamic        DynamicType `config:"dynamic"`
	Index          *bool       `config:"index"`
	DocValues      *bool       `config:"doc_values"`
	CopyTo         string      `config:"copy_to"`
	IgnoreAbove    int         `config:"ignore_above"`
	AliasPath      string      `config:"path"`
	MigrationAlias bool        `config:"migration"`

	ObjectType            string          `config:"object_type"`
	ObjectTypeMappingType string          `config:"object_type_mapping_type"`
	ScalingFactor         int             `config:"scaling_factor"`
	ObjectTypeParams      []ObjectTypeCfg `config:"object_type_params"`

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

	Overwrite bool `config:"overwrite"`
	Path      string
}

// ObjectTypeCfg defines type and configuration of object attributes
type ObjectTypeCfg struct {
	ObjectType            string `config:"object_type"`
	ObjectTypeMappingType string `config:"object_type_mapping_type"`
	ScalingFactor         int    `config:"scaling_factor"`
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

// Validate ensures objectTypeParams are not mixed with top level objectType configuration
func (f *Field) Validate() error {
	if len(f.ObjectTypeParams) == 0 {
		return nil
	}
	if f.ScalingFactor != 0 || f.ObjectTypeMappingType != "" || f.ObjectType != "" {
		return errors.New("mixing top level objectType configuration with array of object type configurations is forbidden")
	}
	return nil
}

func LoadFieldsYaml(path string) (Fields, error) {
	var keys []Field

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

// LoadFields loads fields from a byte array
func LoadFields(f []byte) (Fields, error) {
	var keys []Field

	cfg, err := yaml.NewConfig(f)
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

// GetField returns the field in case it exists
func (f Fields) GetField(key string) *Field {
	keys := strings.Split(key, ".")
	return f.getField(keys)

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

func (f Fields) getField(keys []string) *Field {
	// Nothing to compare anymore
	if len(keys) == 0 {
		return nil
	}

	key := keys[0]
	keys = keys[1:]

	for _, field := range f {
		if field.Name == key {

			if len(field.Fields) > 0 {
				return field.Fields.getField(keys)
			}
			// Last entry in the tree but still more keys
			if len(keys) > 0 {
				return nil
			}

			return &field
		}
	}
	return nil
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

// ConcatFields concatenates two Fields lists into a new list.
// The operation fails if the input definitions define the same keys.
func ConcatFields(a, b Fields) (Fields, error) {
	if len(b) == 0 {
		return a, nil
	}
	if len(a) == 0 {
		return b, nil
	}

	// check for duplicates
	if err := a.conflicts(b); err != nil {
		return nil, err
	}

	// concat a+b into new array
	fields := make(Fields, 0, len(a)+len(b))
	return append(append(fields, a...), b...), nil
}

func (f Fields) conflicts(fields Fields) error {
	var errs multierror.Errors
	for _, key := range fields.GetKeys() {
		keys := strings.Split(key, ".")
		if err := f.canConcat(key, keys); err != nil {
			errs = append(errs, err)
		}
	}
	return errs.Err()
}

// canConcat checks if the given string can be concatenated to the existing fields f
// a key cannot be concatenated if
// - f has a node with name key
// - f has a leaf with key's parent name and the leaf's type is not `object`
func (f Fields) canConcat(k string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	key := keys[0]
	keys = keys[1:]
	for _, field := range f {
		if field.Name != key {
			continue
		}
		// last key to compare
		if len(keys) == 0 {
			return errors.Errorf("fields contain key <%s>", k)
		}
		// last field to compare, only valid if it is of type object
		if len(field.Fields) == 0 {
			if field.Type != "object" {
				return errors.Errorf("fields contain non object node conflicting with key <%s>", k)
			}
		}
		return field.Fields.canConcat(k, keys)
	}
	return nil
}
