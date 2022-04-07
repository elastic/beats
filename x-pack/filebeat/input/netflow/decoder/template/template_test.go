// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package template

import (
	"bytes"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v8/x-pack/filebeat/input/netflow/decoder/fields"
	"github.com/elastic/beats/v8/x-pack/filebeat/input/netflow/decoder/record"
	"github.com/elastic/beats/v8/x-pack/filebeat/input/netflow/decoder/test"
)

func TestTemplate_Apply(t *testing.T) {
	longField := make([]byte, 0x0456)
	for i := range longField {
		longField[i] = byte(i)
	}
	for _, tc := range []struct {
		title    string
		record   Template
		data     []byte
		count    int
		expected []record.Record
		err      error
	}{
		{
			title: "empty template",
			err:   errors.New("empty template"),
		},
		{
			title: "single record guess length and pad",
			record: Template{
				Length: 7,
				Fields: []FieldTemplate{
					{Length: 4, Info: &fields.Field{Name: "sourceIPv4Address", Decoder: fields.Ipv4Address}},
					{Length: 2, Info: &fields.Field{Name: "destinationTransportPort", Decoder: fields.Unsigned16}},
					{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
				},
			},
			data: []byte{
				10, 1, 2, 3, 0x12, 0x34, 59, 0,
			},
			count: 0,
			expected: []record.Record{
				{
					Type: record.Flow,
					Fields: record.Map{
						"sourceIPv4Address":        net.ParseIP("10.1.2.3").To4(),
						"destinationTransportPort": uint64(0x1234),
						"ipClassOfService":         uint64(59),
					},
				},
			},
		},
		{
			title: "two records guess length",
			record: Template{
				Length: 7,
				Fields: []FieldTemplate{
					{Length: 4, Info: &fields.Field{Name: "sourceIPv4Address", Decoder: fields.Ipv4Address}},
					{Length: 2, Info: &fields.Field{Name: "destinationTransportPort", Decoder: fields.Unsigned16}},
					{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
				},
			},
			data: []byte{
				10, 1, 2, 3, 0x12, 0x34, 59,
				127, 0, 0, 1, 0, 80, 12,
			},
			count: 0,
			expected: []record.Record{
				{
					Type: record.Flow,
					Fields: record.Map{
						"sourceIPv4Address":        net.ParseIP("10.1.2.3").To4(),
						"destinationTransportPort": uint64(0x1234),
						"ipClassOfService":         uint64(59),
					},
				},
				{
					Type: record.Flow,
					Fields: record.Map{
						"sourceIPv4Address":        net.ParseIP("127.0.0.1").To4(),
						"destinationTransportPort": uint64(80),
						"ipClassOfService":         uint64(12),
					},
				},
			},
		},
		{
			title: "single record with count",
			record: Template{
				Length: 7,
				Fields: []FieldTemplate{
					{Length: 4, Info: &fields.Field{Name: "sourceIPv4Address", Decoder: fields.Ipv4Address}},
					{Length: 2, Info: &fields.Field{Name: "destinationTransportPort", Decoder: fields.Unsigned16}},
					{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
				},
			},
			data: []byte{
				10, 1, 2, 3, 0x12, 0x34, 59, 0,
			},
			count: 1,
			expected: []record.Record{
				{
					Type: record.Flow,
					Fields: record.Map{
						"sourceIPv4Address":        net.ParseIP("10.1.2.3").To4(),
						"destinationTransportPort": uint64(0x1234),
						"ipClassOfService":         uint64(59),
					},
				},
			},
		},
		{
			title: "single record with count excess",
			record: Template{
				Length: 7,
				Fields: []FieldTemplate{
					{Length: 4, Info: &fields.Field{Name: "sourceIPv4Address", Decoder: fields.Ipv4Address}},
					{Length: 2, Info: &fields.Field{Name: "destinationTransportPort", Decoder: fields.Unsigned16}},
					{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
				},
			},
			data: []byte{
				10, 1, 2, 3, 0x12, 0x34, 59,
				127, 0, 0, 1, 0, 80, 12,
			},
			count: 1,
			expected: []record.Record{
				{
					Type: record.Flow,
					Fields: record.Map{
						"sourceIPv4Address":        net.ParseIP("10.1.2.3").To4(),
						"destinationTransportPort": uint64(0x1234),
						"ipClassOfService":         uint64(59),
					},
				},
			},
		},
		{
			title: "two records with count",
			record: Template{
				Length: 7,
				Fields: []FieldTemplate{
					{Length: 4, Info: &fields.Field{Name: "sourceIPv4Address", Decoder: fields.Ipv4Address}},
					{Length: 2, Info: &fields.Field{Name: "destinationTransportPort", Decoder: fields.Unsigned16}},
					{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
				},
			},
			data: []byte{
				10, 1, 2, 3, 0x12, 0x34, 59,
				127, 0, 0, 1, 0, 80, 12,
			},
			count: 2,
			expected: []record.Record{
				{
					Type: record.Flow,
					Fields: record.Map{
						"sourceIPv4Address":        net.ParseIP("10.1.2.3").To4(),
						"destinationTransportPort": uint64(0x1234),
						"ipClassOfService":         uint64(59),
					},
				},
				{
					Type: record.Flow,
					Fields: record.Map{
						"sourceIPv4Address":        net.ParseIP("127.0.0.1").To4(),
						"destinationTransportPort": uint64(80),
						"ipClassOfService":         uint64(12),
					},
				},
			},
		},
		{
			title: "single record variable length guess count",
			record: Template{
				Length:         6,
				VariableLength: true,
				Fields: []FieldTemplate{
					{Length: 4, Info: &fields.Field{Name: "sourceIPv4Address", Decoder: fields.Ipv4Address}},
					{Length: VariableLength, Info: &fields.Field{Name: "vpnIdentifier", Decoder: fields.OctetArray}},
					{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
				},
			},
			data: []byte{
				10, 1, 2, 3,
				5, 1, 2, 3, 4, 5,
				93,
			},
			count: 0,
			expected: []record.Record{
				{
					Type: record.Flow,
					Fields: record.Map{
						"sourceIPv4Address": net.ParseIP("10.1.2.3").To4(),
						"vpnIdentifier":     []byte{1, 2, 3, 4, 5},
						"ipClassOfService":  uint64(93),
					},
				},
			},
		},
		{
			title: "multiple record variable length guess count",
			record: Template{
				Length:         6,
				VariableLength: true,
				Fields: []FieldTemplate{
					{Length: 4, Info: &fields.Field{Name: "sourceIPv4Address", Decoder: fields.Ipv4Address}},
					{Length: VariableLength, Info: &fields.Field{Name: "vpnIdentifier", Decoder: fields.OctetArray}},
					{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
				},
			},
			data: []byte{
				10, 1, 2, 3,
				5, 1, 2, 3, 4, 5,
				93,
				10, 1, 2, 3,
				2, 123, 234,
				93,
			},
			count: 0,
			expected: []record.Record{
				{
					Type: record.Flow,
					Fields: record.Map{
						"sourceIPv4Address": net.ParseIP("10.1.2.3").To4(),
						"vpnIdentifier":     []byte{1, 2, 3, 4, 5},
						"ipClassOfService":  uint64(93),
					},
				},
				{
					Type: record.Flow,
					Fields: record.Map{
						"sourceIPv4Address": net.ParseIP("10.1.2.3").To4(),
						"vpnIdentifier":     []byte{123, 234},
						"ipClassOfService":  uint64(93),
					},
				},
			},
		},
		{
			title: "long variable length",
			record: Template{
				Length:         6,
				VariableLength: true,
				Fields: []FieldTemplate{
					{Length: 4, Info: &fields.Field{Name: "sourceIPv4Address", Decoder: fields.Ipv4Address}},
					{Length: VariableLength, Info: &fields.Field{Name: "vpnIdentifier", Decoder: fields.OctetArray}},
					{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
				},
			},
			data: append([]byte{10, 1, 2, 3, 0xFF, 0x04, 0x56},
				append(append([]byte{}, longField...), 93, 10, 1, 2, 3, 2, 123, 234, 93)...),
			count: 2,
			expected: []record.Record{
				{
					Type: record.Flow,
					Fields: record.Map{
						"sourceIPv4Address": net.ParseIP("10.1.2.3").To4(),
						"vpnIdentifier":     longField,
						"ipClassOfService":  uint64(93),
					},
				},
				{
					Type: record.Flow,
					Fields: record.Map{
						"sourceIPv4Address": net.ParseIP("10.1.2.3").To4(),
						"vpnIdentifier":     []byte{123, 234},
						"ipClassOfService":  uint64(93),
					},
				},
			},
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			actual, err := tc.record.Apply(bytes.NewBuffer(tc.data), tc.count)
			assert.Equal(t, tc.err, err)
			if assert.Len(t, actual, len(tc.expected)) {
				for i, record := range actual {
					test.AssertRecordsEqual(t, tc.expected[i], record)
				}
			}
		})
	}
}

func TestOptionsTemplate_Apply(t *testing.T) {
	longField := make([]byte, 0x0456)
	for i := range longField {
		longField[i] = byte(i)
	}
	for _, tc := range []struct {
		title    string
		record   Template
		data     []byte
		count    int
		expected []record.Record
		err      error
	}{
		{
			title: "empty template",
			err:   errors.New("empty template"),
		},
		{
			title: "single record guess length and pad",
			record: Template{
				Length:      7,
				ScopeFields: 1,
				IsOptions:   true,
				Fields: []FieldTemplate{
					{Length: 4, Info: &fields.Field{Name: "sourceIPv4Address", Decoder: fields.Ipv4Address}},
					{Length: 2, Info: &fields.Field{Name: "destinationTransportPort", Decoder: fields.Unsigned16}},
					{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
				},
			},
			data: []byte{
				10, 1, 2, 3, 0x12, 0x34, 59, 0,
			},
			count: 0,
			expected: []record.Record{
				{
					Type: record.Options,
					Fields: record.Map{
						"scope": record.Map{
							"sourceIPv4Address": net.ParseIP("10.1.2.3").To4(),
						},
						"options": record.Map{
							"destinationTransportPort": uint64(0x1234),
							"ipClassOfService":         uint64(59),
						},
					},
				},
			},
		},
		{
			title: "two records guess length",
			record: Template{
				Length:      7,
				ScopeFields: 2,
				IsOptions:   true,
				Fields: []FieldTemplate{
					{Length: 4, Info: &fields.Field{Name: "sourceIPv4Address", Decoder: fields.Ipv4Address}},
					{Length: 2, Info: &fields.Field{Name: "destinationTransportPort", Decoder: fields.Unsigned16}},
					{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
				},
			},
			data: []byte{
				10, 1, 2, 3, 0x12, 0x34, 59,
				127, 0, 0, 1, 0, 80, 12,
			},
			count: 0,
			expected: []record.Record{
				{
					Type: record.Options,
					Fields: record.Map{
						"scope": record.Map{
							"sourceIPv4Address":        net.ParseIP("10.1.2.3").To4(),
							"destinationTransportPort": uint64(0x1234),
						},
						"options": record.Map{
							"ipClassOfService": uint64(59),
						},
					},
				},
				{
					Type: record.Options,
					Fields: record.Map{
						"scope": record.Map{
							"sourceIPv4Address":        net.ParseIP("127.0.0.1").To4(),
							"destinationTransportPort": uint64(80),
						},
						"options": record.Map{
							"ipClassOfService": uint64(12),
						},
					},
				},
			},
		},
		{
			title: "single record with count",
			record: Template{
				Length:      7,
				ScopeFields: 3,
				IsOptions:   true,
				Fields: []FieldTemplate{
					{Length: 4, Info: &fields.Field{Name: "sourceIPv4Address", Decoder: fields.Ipv4Address}},
					{Length: 2, Info: &fields.Field{Name: "destinationTransportPort", Decoder: fields.Unsigned16}},
					{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
				},
			},
			data: []byte{
				10, 1, 2, 3, 0x12, 0x34, 59, 0,
			},
			count: 1,
			expected: []record.Record{
				{
					Type: record.Options,
					Fields: record.Map{
						"scope": record.Map{
							"sourceIPv4Address":        net.ParseIP("10.1.2.3").To4(),
							"destinationTransportPort": uint64(0x1234),
							"ipClassOfService":         uint64(59),
						},
						"options": record.Map{},
					},
				},
			},
		},
		{
			title: "single record with count excess",
			record: Template{
				Length:      7,
				ScopeFields: 1,
				IsOptions:   true,
				Fields: []FieldTemplate{
					{Length: 4, Info: &fields.Field{Name: "sourceIPv4Address", Decoder: fields.Ipv4Address}},
					{Length: 2, Info: &fields.Field{Name: "destinationTransportPort", Decoder: fields.Unsigned16}},
					{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
				},
			},
			data: []byte{
				10, 1, 2, 3, 0x12, 0x34, 59,
				127, 0, 0, 1, 0, 80, 12,
			},
			count: 1,
			expected: []record.Record{
				{
					Type: record.Options,
					Fields: record.Map{
						"scope": record.Map{
							"sourceIPv4Address": net.ParseIP("10.1.2.3").To4(),
						},
						"options": record.Map{
							"destinationTransportPort": uint64(0x1234),
							"ipClassOfService":         uint64(59),
						},
					},
				},
			},
		},
		{
			title: "two records with count",
			record: Template{
				Length:      7,
				ScopeFields: 2,
				IsOptions:   true,
				Fields: []FieldTemplate{
					{Length: 4, Info: &fields.Field{Name: "sourceIPv4Address", Decoder: fields.Ipv4Address}},
					{Length: 2, Info: &fields.Field{Name: "destinationTransportPort", Decoder: fields.Unsigned16}},
					{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
				},
			},
			data: []byte{
				10, 1, 2, 3, 0x12, 0x34, 59,
				127, 0, 0, 1, 0, 80, 12,
			},
			count: 2,
			expected: []record.Record{
				{
					Type: record.Options,
					Fields: record.Map{
						"scope": record.Map{
							"sourceIPv4Address":        net.ParseIP("10.1.2.3").To4(),
							"destinationTransportPort": uint64(0x1234),
						},
						"options": record.Map{
							"ipClassOfService": uint64(59),
						},
					},
				},
				{
					Type: record.Options,
					Fields: record.Map{
						"scope": record.Map{
							"sourceIPv4Address":        net.ParseIP("127.0.0.1").To4(),
							"destinationTransportPort": uint64(80),
						},
						"options": record.Map{
							"ipClassOfService": uint64(12),
						},
					},
				},
			},
		},
		{
			title: "single record variable length guess count",
			record: Template{
				Length:         6,
				ScopeFields:    1,
				IsOptions:      true,
				VariableLength: true,
				Fields: []FieldTemplate{
					{Length: 4, Info: &fields.Field{Name: "sourceIPv4Address", Decoder: fields.Ipv4Address}},
					{Length: VariableLength, Info: &fields.Field{Name: "vpnIdentifier", Decoder: fields.OctetArray}},
					{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
				},
			},
			data: []byte{
				10, 1, 2, 3,
				5, 1, 2, 3, 4, 5,
				93,
			},
			count: 0,
			expected: []record.Record{
				{
					Type: record.Options,
					Fields: record.Map{
						"scope": record.Map{
							"sourceIPv4Address": net.ParseIP("10.1.2.3").To4(),
						},
						"options": record.Map{
							"vpnIdentifier":    []byte{1, 2, 3, 4, 5},
							"ipClassOfService": uint64(93),
						},
					},
				},
			},
		},
		{
			title: "multiple record variable length guess count",
			record: Template{
				Length:         6,
				ScopeFields:    1,
				IsOptions:      true,
				VariableLength: true,
				Fields: []FieldTemplate{
					{Length: 4, Info: &fields.Field{Name: "sourceIPv4Address", Decoder: fields.Ipv4Address}},
					{Length: VariableLength, Info: &fields.Field{Name: "vpnIdentifier", Decoder: fields.OctetArray}},
					{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
				},
			},
			data: []byte{
				10, 1, 2, 3,
				5, 1, 2, 3, 4, 5,
				93,
				10, 1, 2, 3,
				2, 123, 234,
				93,
			},
			count: 0,
			expected: []record.Record{
				{
					Type: record.Options,
					Fields: record.Map{
						"scope": record.Map{
							"sourceIPv4Address": net.ParseIP("10.1.2.3").To4(),
						},
						"options": record.Map{
							"vpnIdentifier":    []byte{1, 2, 3, 4, 5},
							"ipClassOfService": uint64(93),
						},
					},
				},
				{
					Type: record.Options,
					Fields: record.Map{
						"scope": record.Map{
							"sourceIPv4Address": net.ParseIP("10.1.2.3").To4(),
						},
						"options": record.Map{
							"vpnIdentifier":    []byte{123, 234},
							"ipClassOfService": uint64(93),
						},
					},
				},
			},
		},
		{
			title: "long variable length",
			record: Template{
				Length:         6,
				VariableLength: true,
				ScopeFields:    2,
				IsOptions:      true,
				Fields: []FieldTemplate{
					{Length: 4, Info: &fields.Field{Name: "sourceIPv4Address", Decoder: fields.Ipv4Address}},
					{Length: VariableLength, Info: &fields.Field{Name: "vpnIdentifier", Decoder: fields.OctetArray}},
					{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
				},
			},
			data: append([]byte{10, 1, 2, 3, 0xFF, 0x04, 0x56},
				append(append([]byte{}, longField...), 93, 10, 1, 2, 3, 2, 123, 234, 93)...),
			count: 2,
			expected: []record.Record{
				{
					Type: record.Options,
					Fields: record.Map{
						"scope": record.Map{
							"sourceIPv4Address": net.ParseIP("10.1.2.3").To4(),
							"vpnIdentifier":     longField,
						},
						"options": record.Map{
							"ipClassOfService": uint64(93),
						},
					},
				},
				{
					Type: record.Options,
					Fields: record.Map{
						"scope": record.Map{
							"sourceIPv4Address": net.ParseIP("10.1.2.3").To4(),
							"vpnIdentifier":     []byte{123, 234},
						},
						"options": record.Map{
							"ipClassOfService": uint64(93),
						},
					},
				},
			},
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			actual, err := tc.record.Apply(bytes.NewBuffer(tc.data), tc.count)
			assert.Equal(t, tc.err, err)
			if assert.Len(t, actual, len(tc.expected)) {
				for i, record := range actual {
					test.AssertRecordsEqual(t, tc.expected[i], record)
				}
			}
		})
	}
}

func TestTemplateEquals(t *testing.T) {
	a := Template{
		ID: 1234,
		Fields: []FieldTemplate{
			{Length: VariableLength, Info: &fields.Field{Name: "wlanSSID", Decoder: fields.String}},
			{Length: 16, Info: &fields.Field{Name: "collectorIPv6Address", Decoder: fields.Ipv6Address}},
		},
		Length:         17,
		VariableLength: true,
		ScopeFields:    0,
	}
	assert.True(t, ValidateTemplate(t, &a))
	b := a
	assert.True(t, AssertTemplateEquals(t, &a, &b))
}
