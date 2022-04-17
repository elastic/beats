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

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/go-ucfg/yaml"
)

// This reflects allowed attributes for field definitions in the fields.yml.
// No logic is put into this data structure.
// The purpose is to enable using different kinds of transformation, on top of the same data structure.
// Current transformation:
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
	Analyzer       Analyzer    `config:"analyzer"`
	SearchAnalyzer Analyzer    `config:"search_analyzer"`
	Norms          bool        `config:"norms"`
	Dynamic        DynamicType `config:"dynamic"`
	Index          *bool       `config:"index"`
	DocValues      *bool       `config:"doc_values"`
	CopyTo         string      `config:"copy_to"`
	IgnoreAbove    int         `config:"ignore_above"`
	AliasPath      string      `config:"path"`
	MigrationAlias bool        `config:"migration"`
	Dimension      *bool       `config:"dimension"`

	// DynamicTemplate controls whether this field represents an explicitly
	// named dynamic template.
	//
	// Such dynamic templates are only suitable for use in dynamic_template
	// parameter in bulk requests or in ingest pipelines, as they will have
	// no path or type match criteria.
	DynamicTemplate bool `config:"dynamic_template"`

	// Unit holds a standard unit for numeric fields: "percent", "byte", or a time unit.
	// See https://www.elastic.co/guide/en/elasticsearch/reference/current/mapping-field-meta.html.
	Unit string `config:"unit"`

	// MetricType holds a standard metric type for numeric fields: "gauge" or "counter".
	// See https://www.elastic.co/guide/en/elasticsearch/reference/current/mapping-field-meta.html.
	MetricType string `config:"metric_type"`

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

	Overwrite    bool  `config:"overwrite"`
	DefaultField *bool `config:"default_field"`
	Path         string
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

type Analyzer struct {
	Name       string
	Definition interface{}
}

func (a *Analyzer) Unpack(v interface{}) error {
	var m common.MapStr
	switch v := v.(type) {
	case string:
		a.Name = v
		return nil
	case common.MapStr:
		m = v
	case map[string]interface{}:
		m = common.MapStr(v)
	default:
		return fmt.Errorf("'%v' is invalid analyzer setting", v)
	}

	if len(m) != 1 {
		return fmt.Errorf("'%v' is invalid analyzer setting", v)
	}
	for a.Name, a.Definition = range m {
		break
	}

	return nil
}

// Validate ensures objectTypeParams are not mixed with top level objectType configuration
func (f *Field) Validate() error {
	if err := f.validateType(); err != nil {
		return errors.Wrapf(err, "incorrect type configuration for field '%s'", f.Name)
	}
	if len(f.ObjectTypeParams) > 0 {
		if f.ScalingFactor != 0 || f.ObjectTypeMappingType != "" || f.ObjectType != "" {
			return errors.New("mixing top level objectType configuration with array of object type configurations is forbidden")
		}
	}
	return nil
}

func (f *Field) validateType() error {
	var allowedFormatters, allowedMetricTypes, allowedUnits []string
	switch strings.ToLower(f.Type) {
	case "text", "keyword", "wildcard", "constant_keyword", "match_only_text":
		allowedFormatters = []string{"string", "url"}
	case "long", "integer", "short", "byte", "double", "float", "half_float", "scaled_float", "histogram":
		allowedFormatters = []string{"string", "url", "bytes", "duration", "number", "percent", "color"}
		allowedMetricTypes = []string{"gauge", "counter"}
		allowedUnits = []string{"percent", "byte", "nanos", "micros", "ms", "s", "m", "h", "d"}
	case "date", "date_nanos":
		allowedFormatters = []string{"string", "url", "date"}
	case "geo_point":
		allowedFormatters = []string{"geo_point"}
	case "date_range":
		allowedFormatters = []string{"date_range"}
	case "boolean", "binary", "ip", "alias", "array", "ip_range":
		// No formatters, metric types, or units allowed.
	case "object":
		if f.DynamicTemplate && (len(f.ObjectTypeParams) > 0 || f.ObjectType != "") {
			// When either ObjectTypeParams or ObjectType are set for an object-type field,
			// libbeat/template will create dynamic templates. It does not make sense to
			// use these with explicit dynamic templates.
			return errors.New("dynamic_template not supported with object_type_params")
		}
		// No further checks for object yet.
		return nil
	case "group", "nested", "flattened":
		// No check for them yet
		return nil
	case "":
		// Module keys, not used as fields
		return nil
	default:
		// There are more types, not being used by beats, to be added if needed
		return fmt.Errorf("unexpected type '%s' for field '%s'", f.Type, f.Name)
	}
	if err := validateAllowedValue(f.Name, "format", f.Format, allowedFormatters); err != nil {
		return err
	}
	if err := validateAllowedValue(f.Name, "metric type", f.MetricType, allowedMetricTypes); err != nil {
		return err
	}
	if err := validateAllowedValue(f.Name, "unit", f.Unit, allowedUnits); err != nil {
		return err
	}
	return nil
}

func validateAllowedValue(fieldName string, propertyName string, propertyValue string, allowedPropertyValues []string) error {
	if propertyValue == "" {
		return nil
	}
	if len(allowedPropertyValues) == 0 {
		return fmt.Errorf("no %s expected for field '%s', found: %s", propertyName, fieldName, propertyValue)
	}
	if !stringsContains(allowedPropertyValues, propertyValue) {
		return fmt.Errorf(
			"unexpected %s '%s' for field '%s', expected one of: %s",
			propertyName, propertyValue, fieldName, strings.Join(allowedPropertyValues, ", "),
		)
	}
	return nil
}

func stringsContains(haystack []string, needle string) bool {
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}

func LoadFieldsYaml(path string) (Fields, error) {
	var keys []Field

	cfg, err := yaml.NewConfigWithFile(path)
	if err != nil {
		return nil, err
	}
	err = cfg.Unpack(&keys)
	if err != nil {
		return nil, err
	}

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
	err = cfg.Unpack(&keys)
	if err != nil {
		return nil, err
	}

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
		if field.ObjectType == "histogram" {
			keys = append(keys, fieldName+".values")
			keys = append(keys, fieldName+".counts")
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
