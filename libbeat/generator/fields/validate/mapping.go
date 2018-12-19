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

package validate

import (
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var ignoreRequired = map[string]struct{}{
	// ECS contains a base group that should be ignored.
	"base.@timestamp": {},
}

// Mapping is the representation of the contents of an Elasticsearch
// index mapping.
type Mapping struct {
	Fields   map[string]Field
	Required []string
}

// Field is the representation of an Elasticsearch mapping field or property.
type Field struct {
	Type string
}

// NewMapping extracts a mapping from a raw yaml format.
func NewMapping(fieldsYAML []byte) (Mapping, error) {
	result := Mapping{
		Fields: make(map[string]Field),
	}

	var fields interface{}
	if err := yaml.Unmarshal(fieldsYAML, &fields); err != nil {
		return result, errors.Wrap(err, "decoding fields YAML")
	}
	subFields, ok := fields.([]interface{})
	if !ok {
		return result, errors.Errorf("top-level fields is not an array, but %T", fields)
	}
	for _, subField := range subFields {
		if err := recursiveFattenFields(subField, "", &result, ""); err != nil {
			return result, err
		}
	}
	return result, nil
}

// Validate takes a document map and validates it against the mapping.
func (m *Mapping) Validate(dict map[string]interface{}) error {
	seen, err := m.validateFields(dict, "")
	if err != nil {
		return err
	}
	docFields := make(map[string]struct{})
	for _, field := range seen {
		docFields[field] = struct{}{}
	}
	for _, required := range m.Required {
		if _, found := docFields[required]; !found {
			return errors.Errorf("required field '%s' not found", required)
		}
	}
	return nil
}

func (m *Mapping) validateFields(dict map[string]interface{}, prefix Prefix) (seen []string, err error) {
	for key, value := range dict {
		name := prefix.Append(key)
		field, found := m.Fields[name.String()]
		if !found {
			return nil, errors.Errorf("field %s not found in mapping", name.String())
		}
		dicts, err := typeCheck(value, field.Type)
		if err != nil {
			return nil, errors.Wrapf(err, "field %s does not match expected type %s", name.String(), field.Type)
		}
		for _, dict := range dicts {
			s, err := m.validateFields(dict, name)
			if err != nil {
				return nil, err
			}
			seen = append(seen, s...)
		}
		seen = append(seen, name.String())
	}
	return seen, nil
}

func (m *Mapping) addField(path string, field Field, required bool) error {
	err := m.storeField(path, field)
	if err != nil {
		return err
	}
	if required {
		if _, found := ignoreRequired[path]; !found {
			m.Required = append(m.Required, path)
		}
	}
	for {
		dot := strings.LastIndexByte(path, '.')
		if dot == -1 {
			break
		}
		path = path[:dot]
		err := m.storeField(path, Field{Type: "group"})
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Mapping) storeField(path string, field Field) error {
	if prev, found := m.Fields[path]; !found {
		m.Fields[path] = field
	} else {
		// Allow groups to be specified more than once
		if prev.Type != "group" || field.Type != "group" {
			return errors.Errorf("duplicate field %s (types %s and %s)", path, prev.Type, field.Type)
		}
	}
	return nil
}

func recursiveFattenFields(fields interface{}, prefix Prefix, mapping *Mapping, key string) error {
	dict, ok := fields.(map[interface{}]interface{})
	if !ok {
		return errors.Errorf("fields entry [%s](%s) is not a dictionary", key, prefix)
	}
	keyIf, hasKey := dict["key"]
	nameIf, hasName := dict["name"]
	fieldsIf, hasFields := dict["fields"]
	typIf, hasType := dict["type"]
	requiredIf, hasRequired := dict["required"]

	var name, typ string
	var required bool

	if hasKey {
		newKey, ok := keyIf.(string)
		if !ok {
			return errors.Errorf("a 'key' field is not of type string, but %T (value=%v)", keyIf, keyIf)
		}
		if len(key) > 0 {
			return errors.Errorf("unexpected 'key' field in [%s](%s). Keys can only be defined at top level", key, prefix)
		}
		key = newKey
	} else {
		if len(key) == 0 {
			return errors.Errorf("found top-level fields entry without a 'key' field")
		}
	}

	if hasName {
		name, ok = nameIf.(string)
		if !ok {
			return errors.Errorf("a field in [%s](%s) has a 'name' entry of unexpected type (type=%T value=%v)", key, prefix, nameIf, nameIf)
		}
		prefix = prefix.Append(name)
	} else {
		if !hasKey {
			if _, hasRelease := dict["release"]; hasRelease {
				// Ignore fields that have no name or key, but a release. Used in metricbeat to document some modules.
				return nil
			}
			return errors.Errorf("field [%s](%s) has a sub-field without 'name' nor 'key'", key, prefix)
		}
	}

	if hasType {
		typ, ok = typIf.(string)
		if !ok {
			return errors.Errorf("field [%s](%s) has a 'type' entry of unexpected type (type=%T value=%v)", key, prefix, nameIf, nameIf)
		}
		if typ == "object" {
			typ = "group"
		}
	}

	if hasRequired {
		required, ok = requiredIf.(bool)
		if !ok {
			return errors.Errorf("field [%s](%s) has 'required' property but is not a boolean, but %T (value=%v)", key, prefix, requiredIf, requiredIf)
		}
	}

	if !hasFields && typ != "group" {
		// Parse a leaf field (not a group)

		if !hasType {
			typ = "keyword"
		}

		path := prefix.String()
		if err := mapping.addField(path, Field{Type: typ}, required); err != nil {
			return errors.Wrapf(err, "adding field [%s](%s)", key, path)
		}
		return nil
	}

	// Parse a group

	if hasType && typ != "group" {
		return errors.Errorf("field [%s](%s) has a 'fields' tag but type is not group (type=%s)", key, prefix, typ)
	}
	if !hasType {
		typ = "group"
	}

	if hasName {
		path := prefix.String()
		if err := mapping.addField(path, Field{Type: typ}, required); err != nil {
			return errors.Wrapf(err, "adding field [%s](%s)", key, path)
		}
	}

	if fieldsIf != nil {
		innerFields, ok := fieldsIf.([]interface{})
		if !ok {
			return errors.Errorf("field [%s](%s) has a 'fields' tag of unexpected type (type=%T value=%v)", key, prefix, nameIf, nameIf)
		}
		for _, field := range innerFields {
			if err := recursiveFattenFields(field, prefix, mapping, key); err != nil {
				return err
			}

		}
	}
	return nil
}
