// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package template

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"

	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/fields"
	"github.com/menderesk/beats/v7/x-pack/filebeat/input/netflow/decoder/record"
)

const (
	VariableLength uint16 = 0xffff
)

var ErrEmptyTemplate = errors.New("empty template")

type Template struct {
	ID             uint16
	Fields         []FieldTemplate
	Length         int
	VariableLength bool
	ScopeFields    int
	// IsOptions signals that this is an options template. Previously
	// ScopeFields>0 was used for this, but that's unreliable under v9.
	IsOptions bool
}

type FieldTemplate struct {
	Length uint16
	Info   *fields.Field
}

func PopulateFieldMap(dest record.Map, fields []FieldTemplate, variableLength bool, buffer *bytes.Buffer) error {
	for _, field := range fields {
		length := field.Length
		if variableLength && length == VariableLength {
			tmp := buffer.Next(1)
			if len(tmp) != 1 {
				return io.EOF
			}
			length = uint16(tmp[0])
			if length == 255 {
				tmp = buffer.Next(2)
				if len(tmp) != 2 {
					return io.EOF
				}
				length = binary.BigEndian.Uint16(tmp)
			}
		}
		raw := buffer.Next(int(length))
		if len(raw) != int(length) {
			return io.EOF
		}
		if fieldInfo := field.Info; fieldInfo != nil {
			value, err := fieldInfo.Decoder.Decode(raw)
			if err != nil {
				continue
			}
			dest[fieldInfo.Name] = value
		}
	}
	return nil
}

func (t *Template) Apply(data *bytes.Buffer, n int) ([]record.Record, error) {
	if t.Length == 0 {
		return nil, ErrEmptyTemplate
	}
	if n == 0 {
		n = data.Len() / t.Length
	}
	limit, alloc := n, n
	if t.VariableLength {
		limit = math.MaxInt16
		alloc = n
		if alloc > 16 {
			alloc = 16
		}
	}
	makeFn := t.makeFlow
	if t.IsOptions {
		makeFn = t.makeOptions
	}
	events := make([]record.Record, 0, alloc)
	for i := 0; i < limit; i++ {
		event, err := makeFn(data)
		if err != nil {
			if err == io.EOF && t.VariableLength {
				break
			}
			return events, err
		}
		events = append(events, event)
	}
	return events, nil
}

func (t *Template) makeFlow(data *bytes.Buffer) (ev record.Record, err error) {
	ev = record.Record{
		Type:   record.Flow,
		Fields: record.Map{},
	}
	if err = PopulateFieldMap(ev.Fields, t.Fields, t.VariableLength, data); err != nil {
		return ev, err
	}
	return ev, nil
}

func (t *Template) makeOptions(data *bytes.Buffer) (ev record.Record, err error) {
	scope := record.Map{}
	options := record.Map{}
	ev = record.Record{
		Type: record.Options,
		Fields: record.Map{
			"scope":   scope,
			"options": options,
		},
	}
	if err = PopulateFieldMap(scope, t.Fields[:t.ScopeFields], t.VariableLength, data); err != nil {
		return ev, err
	}
	if err = PopulateFieldMap(options, t.Fields[t.ScopeFields:], t.VariableLength, data); err != nil {
		return ev, err
	}
	return ev, nil
}
