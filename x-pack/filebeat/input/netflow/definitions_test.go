// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package netflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"

	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/fields"
)

func TestLoadFieldDefinitions(t *testing.T) {
	for _, testCase := range []struct {
		title, yaml string
		expected    fields.FieldDict
	}{
		{
			title: "IPFIX definitions",
			yaml: `
1234:
  0:
    - :skip
  7:
  - 4
  - :rawField
  11:
  - :ip4_addr
  - :ip4_field
  33:
  - :ip6_addr
  - :ipv6_field
  42:
  - :int15
  - :dword_field
0x29a:
  128:
  - :mac_addr
  - :mac_field
  999:
  - :string
  - :name
`,
			expected: fields.FieldDict{
				fields.Key{EnterpriseID: 1234, FieldID: 7}:  &fields.Field{Name: "rawField", Decoder: fields.Unsigned32},
				fields.Key{EnterpriseID: 1234, FieldID: 11}: &fields.Field{Name: "ip4_field", Decoder: fields.Ipv4Address},
				fields.Key{EnterpriseID: 1234, FieldID: 33}: &fields.Field{Name: "ipv6_field", Decoder: fields.Ipv6Address},
				fields.Key{EnterpriseID: 1234, FieldID: 42}: &fields.Field{Name: "dword_field", Decoder: fields.SignedDecoder(15)},
				fields.Key{EnterpriseID: 666, FieldID: 128}: &fields.Field{Name: "mac_field", Decoder: fields.MacAddress},
				fields.Key{EnterpriseID: 666, FieldID: 999}: &fields.Field{Name: "name", Decoder: fields.String},
			},
		},
		{
			title: "NetFlow definitions",
			yaml: `
1:
 - :double
 - MyDouble
2:
 - :float
 - :SomeFloat
3:
 - skip
4:
 - mac_addr
 - :peerMac
5:
 - 3
 - :rgbColor
6:
 - :octet_array
 - :bunchBytes
7:
 - :skip
8:
 - :forwarding_status
 - :status
`,
			expected: fields.FieldDict{
				fields.Key{EnterpriseID: 0, FieldID: 1}: &fields.Field{Name: "MyDouble", Decoder: fields.Float64},
				fields.Key{EnterpriseID: 0, FieldID: 2}: &fields.Field{Name: "SomeFloat", Decoder: fields.Float32},
				fields.Key{EnterpriseID: 0, FieldID: 4}: &fields.Field{Name: "peerMac", Decoder: fields.MacAddress},
				fields.Key{EnterpriseID: 0, FieldID: 5}: &fields.Field{Name: "rgbColor", Decoder: fields.UnsignedDecoder(24)},
				fields.Key{EnterpriseID: 0, FieldID: 6}: &fields.Field{Name: "bunchBytes", Decoder: fields.OctetArray},
				fields.Key{EnterpriseID: 0, FieldID: 8}: &fields.Field{Name: "status", Decoder: fields.UnsupportedDecoder{}},
			},
		},
	} {
		t.Run(testCase.title, func(t *testing.T) {
			var tree interface{}
			if err := yaml.Unmarshal([]byte(testCase.yaml), &tree); err != nil {
				t.Fatal(err)
			}
			defs, err := LoadFieldDefinitions(tree)
			if !assert.NoError(t, err) {
				t.Fatal(err)
			}
			if !assert.Len(t, defs, len(testCase.expected)) {
				t.FailNow()
			}
			for key, value := range testCase.expected {
				assert.Contains(t, defs, key)
				assert.Equal(t, *value, *defs[key])
			}
		})
	}
}
