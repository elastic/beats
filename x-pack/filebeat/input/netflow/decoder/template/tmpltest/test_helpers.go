// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Package tmpltest provides shared template testing functions for netflow templates.
package tmpltest

import (
	"fmt"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v9/x-pack/filebeat/input/netflow/decoder/fields"
	"github.com/elastic/beats/v9/x-pack/filebeat/input/netflow/decoder/template"
)

var (
	decoderByName = map[string]fields.Decoder{}
	once          sync.Once
)

func buildDecoderByNameMap() {
	for _, value := range fields.GlobalFields {
		decoderByName[value.Name] = value.Decoder
	}
}

func ValidateTemplate(t testing.TB, tmpl *template.Template) bool {
	once.Do(buildDecoderByNameMap)

	sum := 0
	seen := make(map[string]bool)
	for idx, field := range tmpl.Fields {
		isVariable := tmpl.VariableLength && field.Length == template.VariableLength
		if !isVariable {
			sum += int(field.Length)
		} else {
			sum += 1
		}
		if field.Info != nil {
			msg := fmt.Sprintf("field[%d]: \"%s\"", idx, field.Info.Name)
			if !assert.NotNil(t, field.Info.Decoder, msg) || !isVariable && (!assert.True(t, field.Info.Decoder.MinLength() <= field.Length, msg) ||
				!assert.True(t, field.Info.Decoder.MaxLength() >= field.Length, msg)) {
				return false
			}
			if !assert.False(t, seen[field.Info.Name], msg) {
				return false
			}
			seen[field.Info.Name] = true
			knownDecoder, found := decoderByName[field.Info.Name]
			if !assert.True(t, found, msg) ||
				!assert.Equal(t, knownDecoder, field.Info.Decoder, msg) {
				return false
			}
		}
	}
	return assert.Equal(t, tmpl.Length, sum) &&
		assert.Equal(t, 0, tmpl.ScopeFields)
}

func AssertFieldsEquals(t testing.TB, expected []template.FieldTemplate, actual []template.FieldTemplate) (succeeded bool) {
	if succeeded = assert.Len(t, actual, len(expected)); succeeded {
		for idx := range expected {
			succeeded = assert.Equal(t, expected[idx].Length, actual[idx].Length, strconv.Itoa(idx)) && succeeded
			succeeded = assert.Equal(t, expected[idx].Info, actual[idx].Info, strconv.Itoa(idx)) && succeeded
		}
	}
	return
}

func AssertTemplateEquals(t testing.TB, expected *template.Template, actual *template.Template) bool {
	if expected == nil && actual == nil {
		return true
	}
	if !assert.True(t, (expected == nil) == (actual == nil)) {
		return false
	}
	assert.Equal(t, expected.VariableLength, actual.VariableLength)
	assert.Equal(t, expected.Length, actual.Length)
	assert.Equal(t, expected.ScopeFields, actual.ScopeFields)
	assert.Equal(t, actual.ID, actual.ID)
	return AssertFieldsEquals(t, actual.Fields, actual.Fields)
}
