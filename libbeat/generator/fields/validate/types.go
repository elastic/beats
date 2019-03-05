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
	"errors"
	"time"
)

var typeValidator = map[string]func(interface{}) bool{
	"long":    longValidator,
	"integer": integerValidator,
	"short":   shortValidator,
	"byte":    byteValidator,
	"double":  doubleValidator,
	"float":   floatValidator,
	"keyword": keywordValidator,
	"text":    textValidator,
	"date":    dateValidator,
	"boolean": booleanValidator,

	// TODO validate this types
	//"half_float":   halfFloatValidator,
	//"scaled_float": scaledFloatValidator,
	//"nested"
	//"ip"
	//"binary":       binaryValidator,
	//"geo_point":...
}

var validTimeFormats = []string{time.RFC3339Nano, time.RFC3339}

// ErrCannotConvert is returned when the document value cannot be converted
// to the expected Elasticsearch datatype.
var ErrCannotConvert = errors.New("no conversion possible")

// ErrNotSupported is returned when an Elasticsearch type is not (yet) supported.
var ErrNotSupported = errors.New("type not supported")

// TODO: numeric validators. The default JSON parser seems to parse all
// numbers into a float64.
func longValidator(value interface{}) bool {
	switch value.(type) {
	case int, int64, float64:
		return true
	}
	return false
}

func integerValidator(value interface{}) bool {
	if val, ok := value.(int); ok {
		return val >= -0x80000000 && val <= 0x7fffffff
	}
	return false
}

func shortValidator(value interface{}) bool {
	if val, ok := value.(int); ok {
		return val >= -0x8000 && val <= 0x7fff
	}
	return false
}

func byteValidator(value interface{}) bool {
	if val, ok := value.(int); ok {
		return val >= -128 && val <= 127
	}
	return false
}

func doubleValidator(value interface{}) bool {
	_, ok := value.(float64)
	return ok
}

func floatValidator(value interface{}) bool {
	if val, ok := value.(float64); ok {
		// Unsure of this because of rounding when serialized to JSON
		return float64(float32(val)) == val
	}
	return false
}

func keywordValidator(value interface{}) bool {
	// Everything is a keyword
	return true
}

func textValidator(value interface{}) bool {
	_, ok := value.(string)
	return ok
}

func dateValidator(value interface{}) bool {
	s, ok := value.(string)
	if !ok {
		return false
	}
	for _, format := range validTimeFormats {
		if _, err := time.Parse(format, s); err == nil {
			return true
		}
	}
	return false
}

func booleanValidator(value interface{}) bool {
	_, ok := value.(bool)
	return ok
}

func typeCheck(value interface{}, expected string) (dicts []map[string]interface{}, err error) {
	switch v := value.(type) {
	case map[string]interface{}:
		if expected != "group" {
			return nil, ErrCannotConvert
		}
		return append(dicts, v), nil

	case []map[string]interface{}:
		if expected != "group" {
			return nil, ErrCannotConvert
		}
		return v, nil

	case []interface{}:
		for _, item := range v {
			if _, err := typeCheck(item, expected); err != nil {
				return nil, err
			}
		}
		return nil, nil

	default:
		if validator, ok := typeValidator[expected]; ok {
			if validator(value) {
				return nil, nil
			}
			return nil, ErrCannotConvert
		}
		return nil, ErrNotSupported
	}
}
