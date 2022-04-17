// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package ipfix

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/fields"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/template"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/test"
	v9 "github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/v9"
)

func TestDecoderV9_ReadFieldDefinition(t *testing.T) {
	decoder := DecoderIPFIX{}
	for _, tc := range []struct {
		title  string
		raw    []byte
		field  fields.Key
		length uint16
		err    error
	}{
		{
			title: "valid field",
			raw: []byte{
				1, 2, 3, 4,
			},
			field:  fields.Key{FieldID: 0x0102},
			length: 0x0304,
		},
		{
			title: "short field",
			raw: []byte{
				1, 2, 3,
			},
			err: io.EOF,
		},
		{
			title: "enterprise id",
			raw: []byte{
				0x80, 1, 0, 4, 0x11, 0x22, 0x33, 0x44,
			},
			field:  fields.Key{EnterpriseID: 0x11223344, FieldID: 1},
			length: 4,
		},
		{
			title: "max length",
			raw: []byte{
				0x12, 0x34, 0xff, 0xff,
			},
			field:  fields.Key{FieldID: 0x1234},
			length: 0xffff,
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			field, length, err := decoder.ReadFieldDefinition(bytes.NewBuffer(tc.raw))
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.field, field)
			assert.Equal(t, tc.length, length)
		})
	}
}

func TestDecoderV9_ReadFields(t *testing.T) {
	decoder := DecoderIPFIX{}
	for _, tc := range []struct {
		title    string
		packet   []uint16
		count    int
		expected template.Template
		err      error
	}{
		{
			title: "valid fields",
			packet: []uint16{
				1, 4,
				5, 1,
				14, 2,
			},
			count: 3,
			expected: template.Template{
				Fields: []template.FieldTemplate{
					{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
					{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
					{Length: 2, Info: &fields.Field{Name: "egressInterface", Decoder: fields.Unsigned32}},
				},
				Length: 7,
			},
		},
		{
			title: "length out of bounds",
			packet: []uint16{
				1, 4,
				5, 11,
				14, 2,
			},
			count: 3,
			expected: template.Template{
				Fields: []template.FieldTemplate{
					{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
					{Length: 11},
					{Length: 2, Info: &fields.Field{Name: "egressInterface", Decoder: fields.Unsigned32}},
				},
				Length: 17,
			},
		},
		{
			title: "enterprise ID",
			packet: []uint16{
				1, 4,
				5, 1,
				0x8000 | 128, 4,
				0, 5951,
			},
			count: 3,
			expected: template.Template{
				Fields: []template.FieldTemplate{
					{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
					{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
					{Length: 4, Info: &fields.Field{Name: "netscalerRoundTripTime", Decoder: fields.Unsigned32}},
				},
				Length: 9,
			},
		},
		{
			title: "EOF",
			packet: []uint16{
				1, 4,
				5, 1,
			},
			count: 3,
			err:   io.EOF,
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			raw := test.MakePacket(tc.packet)
			record, err := v9.ReadFields(decoder, raw, tc.count)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.expected.Length, record.Length)
			assert.Equal(t, tc.expected.VariableLength, record.VariableLength)
			assert.Equal(t, tc.expected.ID, record.ID)
			template.AssertFieldsEquals(t, tc.expected.Fields, record.Fields)
		})
	}
}

func TestReadOptionsTemplateFlowSet(t *testing.T) {
	decoder := DecoderIPFIX{}
	for _, tc := range []struct {
		title    string
		packet   []uint16
		expected []*template.Template
		err      error
	}{
		{
			title: "valid fields",
			packet: []uint16{
				999, 3, 1,
				1, 4,
				5, 1,
				14, 2,
				998, 1, 1,
				16, 4,
			},
			expected: []*template.Template{
				{
					ID:          999,
					Length:      7,
					ScopeFields: 1,
					Fields: []template.FieldTemplate{
						{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
						{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
						{Length: 2, Info: &fields.Field{Name: "egressInterface", Decoder: fields.Unsigned32}},
					},
				},
				{
					ID:          998,
					Length:      4,
					ScopeFields: 1,
					Fields: []template.FieldTemplate{
						{Length: 4, Info: &fields.Field{Name: "bgpSourceAsNumber", Decoder: fields.Unsigned32}},
					},
				},
			},
		},
		{
			title: "variable length",
			packet: []uint16{
				999, 3, 2,
				1, 0xFFFF,
				5, 1,
				14, 2,
				998, 1, 1,
				16, 4,
			},
			expected: []*template.Template{
				{
					ID:             999,
					Length:         4,
					VariableLength: true,
					ScopeFields:    2,
					Fields: []template.FieldTemplate{
						{Length: 0xFFFF, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
						{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
						{Length: 2, Info: &fields.Field{Name: "egressInterface", Decoder: fields.Unsigned32}},
					},
				},
				{
					ID:          998,
					Length:      4,
					ScopeFields: 1,
					Fields: []template.FieldTemplate{
						{Length: 4, Info: &fields.Field{Name: "bgpSourceAsNumber", Decoder: fields.Unsigned32}},
					},
				},
			},
		},
		{
			title: "EOF",
			packet: []uint16{
				999, 44, 8,
				1, 4,
				5, 1,
				14, 2,
				1, 4, 0,
				16, 4,
			},
			err: io.EOF,
		},
		{
			title: "bad length",
			packet: []uint16{
				999, 1, 3,
				1, 4,
				5, 1,
				14, 2,
				1111, 1, 1,
				16, 4,
				0, 0, 0, 0, 0, 0, 0, 0,
			},
			err: errors.New("wrong counts in options template flowset: scope=3 total=1"),
		},
		{
			title: "invalid template ID",
			packet: []uint16{
				999, 3, 2,
				1, 4,
				5, 1,
				14, 2,
				1, 4, 2,
				16, 4,
				0, 0, 0, 0, 0, 0, 0, 0,
			},
			err: errors.New("invalid template id"),
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			raw := test.MakePacket(tc.packet)
			templates, err := decoder.ReadOptionsTemplateFlowSet(raw)
			assert.Equal(t, tc.err, err)
			if assert.Len(t, templates, len(tc.expected)) {
				for idx := range tc.expected {
					template.AssertTemplateEquals(t, tc.expected[idx], templates[idx])
				}
			}
		})
	}
}

func TestReadRecordTemplateFlowSet(t *testing.T) {
	decoder := DecoderIPFIX{}
	for _, tc := range []struct {
		title    string
		packet   []uint16
		expected []*template.Template
		err      error
	}{
		{
			title: "valid fields",
			packet: []uint16{
				999, 3,
				1, 4,
				5, 1,
				14, 2,
				998, 1,
				16, 4,
			},
			expected: []*template.Template{
				{
					ID:     999,
					Length: 7,
					Fields: []template.FieldTemplate{
						{Length: 4, Info: &fields.Field{Name: "octetDeltaCount", Decoder: fields.Unsigned64}},
						{Length: 1, Info: &fields.Field{Name: "ipClassOfService", Decoder: fields.Unsigned8}},
						{Length: 2, Info: &fields.Field{Name: "egressInterface", Decoder: fields.Unsigned32}},
					},
				},
				{
					ID:     998,
					Length: 4,
					Fields: []template.FieldTemplate{
						{Length: 4, Info: &fields.Field{Name: "bgpSourceAsNumber", Decoder: fields.Unsigned32}},
					},
				},
			},
		},
		{
			title: "EOF",
			packet: []uint16{
				999, 44,
				1, 4,
				5, 1,
				14, 2,
				1, 4,
				16, 4,
			},
			err: io.EOF,
		},
		{
			title: "bad ID",
			packet: []uint16{
				99, 6,
				1, 4,
				5, 1,
				14, 2,
			},
			err: errors.New("invalid template id"),
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			raw := test.MakePacket(tc.packet)
			templates, err := v9.ReadTemplateFlowSet(decoder, raw)
			assert.Equal(t, tc.err, err)
			if assert.Len(t, templates, len(tc.expected)) {
				for idx := range tc.expected {
					template.AssertTemplateEquals(t, tc.expected[idx], templates[idx])
				}
			}
		})
	}
}
