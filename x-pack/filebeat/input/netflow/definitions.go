// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package netflow

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/netflow/decoder/fields"
)

var logstashName2Decoder = map[string]fields.Decoder{
	"double":                  fields.Float64,
	"float":                   fields.Float32,
	"int8":                    fields.Signed8,
	"int15":                   fields.SignedDecoder(15),
	"int16":                   fields.Signed16,
	"int24":                   fields.SignedDecoder(24),
	"int32":                   fields.Signed32,
	"int64":                   fields.Signed64,
	"ip4_addr":                fields.Ipv4Address,
	"ip4addr":                 fields.Ipv4Address,
	"ip6_addr":                fields.Ipv6Address,
	"ip6addr":                 fields.Ipv6Address,
	"mac_addr":                fields.MacAddress,
	"macaddr":                 fields.MacAddress,
	"string":                  fields.String,
	"uint8":                   fields.Unsigned8,
	"uint15":                  fields.UnsignedDecoder(15),
	"uint16":                  fields.Unsigned16,
	"uint24":                  fields.UnsignedDecoder(24),
	"uint32":                  fields.Unsigned32,
	"uint64":                  fields.Unsigned64,
	"octet_array":             fields.OctetArray,
	"octetarray":              fields.OctetArray,
	"acl_id_asa":              fields.ACLID,
	"mpls_label_stack_octets": fields.UnsupportedDecoder{},
	"application_id":          fields.UnsupportedDecoder{},
	"forwarding_status":       fields.UnsupportedDecoder{},
}

// LoadFieldDefinitions takes a parsed YAML tree from a Logstash
// Netflow or IPFIX custom fields format and converts it to a FieldDict.
func LoadFieldDefinitions(yaml interface{}) (defs fields.FieldDict, err error) {
	tree, ok := yaml.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid custom fields definition format: expected a mapping of integer keys. Got %T", yaml)
	}
	if len(tree) == 0 {
		return nil, nil
	}
	isIPFIX, err := fieldsAreIPFIX(tree)
	if err != nil {
		return nil, err
	}
	defs = fields.FieldDict{}
	if !isIPFIX {
		if err := loadFields(tree, 0, defs); err != nil {
			return nil, fmt.Errorf("failed to load NetFlow fields: %w", err)
		}
		return defs, nil
	}
	for pemI, fields := range tree {
		pem, err := toInt(pemI)
		if err != nil {
			return nil, err
		}
		if !fits(pem, 0, math.MaxUint32) {
			return nil, fmt.Errorf("PEM %d out of uint32 range", pem)
		}
		tree, ok := fields.(map[interface{}]interface{})
		if !ok {
			return nil, fmt.Errorf("IPFIX fields for pem=%d malformed", pem)
		}
		if err := loadFields(tree, uint32(pem), defs); err != nil {
			return nil, fmt.Errorf("failed to load IPFIX fields for pem=%d: %w", pem, err)
		}
	}
	return defs, nil
}

// LoadFieldDefinitionsFromFile takes the path to a YAML file in Logstash
// Netflow or IPFIX custom fields format and converts it to a FieldDict.
func LoadFieldDefinitionsFromFile(path string) (defs fields.FieldDict, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	contents, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	var tree interface{}
	if err := yaml.Unmarshal(contents, &tree); err != nil {
		return nil, fmt.Errorf("unable to parse YAML: %w", err)
	}
	return LoadFieldDefinitions(tree)
}

func fits(value, min, max int64) bool {
	return value >= min && value <= max
}

func trimColon(s string) string {
	if len(s) > 0 && s[0] == ':' {
		return s[1:]
	}
	return s
}

func toInt(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 0, 64)
	}
	return 0, fmt.Errorf("value %v cannot be converted to int", value)
}

func loadFields(def map[interface{}]interface{}, pem uint32, dest fields.FieldDict) error {
	for keyI, iface := range def {
		fieldID, err := toInt(keyI)
		if err != nil {
			return err
		}
		if !fits(fieldID, 0, math.MaxUint16) {
			return fmt.Errorf("field ID %d out of range uint16", fieldID)
		}
		list, ok := iface.([]interface{})
		if !ok {
			return fmt.Errorf("field ID %d is not a list", fieldID)
		}
		bad := true
		var fieldType, fieldName string
		switch len(list) {
		case 2:
			switch v := list[0].(type) {
			case string:
				fieldType = trimColon(v)
			case int:
				if v == 0 {
					v = 4
				}
				fieldType = fmt.Sprintf("uint%d", v*8)
			}
			if name, ok := list[1].(string); ok {
				fieldName = trimColon(name)
				bad = len(fieldType) == 0 || len(fieldName) == 0
			}
		case 1:
			str, ok := list[0].(string)
			if ok && trimColon(str) == "skip" {
				continue
			}
		}
		if bad {
			return fmt.Errorf("bad field ID %d: should have two items (type, name) or one (:skip) (Got %+v)", fieldID, list)
		}
		key := fields.Key{
			EnterpriseID: pem,
			FieldID:      uint16(fieldID),
		}
		if _, exists := dest[key]; exists {
			return fmt.Errorf("repeated field ID %d", fieldID)
		}
		decoder, found := logstashName2Decoder[fieldType]
		if !found {
			return fmt.Errorf("field ID %d has unknown type %s", fieldID, fieldType)
		}
		dest[key] = &fields.Field{
			Name:    fieldName,
			Decoder: decoder,
		}
	}
	return nil
}

func fieldsAreIPFIX(tree map[interface{}]interface{}) (bool, error) {
	if len(tree) == 0 {
		return false, errors.New("custom fields definition is empty")
	}
	var seenList, seenMap bool
	for key, value := range tree {
		var msg string
		switch v := value.(type) {
		case map[interface{}]interface{}:
			seenMap = true
			if seenList {
				msg = "expected IPFIX map of fields"
			}
		case []interface{}:
			seenList = true
			if seenMap {
				msg = "expected NetFlow single field definition"
			}
		default:
			msg = fmt.Sprintf("unexpected format, got %T", v)
		}
		if len(msg) > 0 {
			return false, fmt.Errorf("inconsistent custom fields definition format: %s at key %v",
				msg, key)
		}
	}
	return seenMap, nil
}
