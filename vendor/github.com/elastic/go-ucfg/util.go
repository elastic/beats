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

package ucfg

import (
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

type tagOptions struct {
	squash      bool
	ignore      bool
	cfgHandling configHandling
}

// configHandling configures the operation to execute if we merge into a struct
// field that holds an unpacked config object.
type configHandling uint8

const (
	cfgDefaultHandling configHandling = iota
	cfgMergeValues
	cfgReplaceValue
	cfgArrAppend
	cfgArrPrepend
)

var noTagOpts = tagOptions{}

func parseTags(tag string) (string, tagOptions) {
	s := strings.Split(tag, ",")
	opts := tagOptions{}
	for _, opt := range s[1:] {
		switch opt {
		case "squash", "inline":
			opts.squash = true
		case "ignore":
			opts.ignore = true
		case "merge":
			opts.cfgHandling = cfgMergeValues
		case "replace":
			opts.cfgHandling = cfgReplaceValue
		case "append":
			opts.cfgHandling = cfgArrAppend
		case "prepend":
			opts.cfgHandling = cfgArrPrepend
		}
	}
	return s[0], opts
}

func fieldName(tagName, structName string) string {
	if tagName != "" {
		return tagName
	}
	return strings.ToLower(structName)
}

func chaseValueInterfaces(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Interface && !v.IsNil() {
		v = v.Elem()
	}
	return v
}

func chaseValuePointers(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	return v
}

func chaseValue(v reflect.Value) reflect.Value {
	for (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) && !v.IsNil() {
		v = v.Elem()
	}
	return v
}

func chaseTypePointers(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// tryTConfig tries to convert input value into addressable Config by converting
// to *Config first. If value is convertible to Config, but not addressable a new
// value is allocated in order to guarantee returned value of type Config is
// addressable. Returns false if type value is not convertible to TConfig.
func tryTConfig(value reflect.Value) (reflect.Value, bool) {
	v := chaseValue(value)
	t := v.Type()

	if t == tConfig {
		v := pointerize(tConfigPtr, tConfig, v)
		return v.Elem(), true
	}

	if !t.ConvertibleTo(tConfig) {
		return reflect.Value{}, false
	}

	v = pointerize(reflect.PtrTo(v.Type()), v.Type(), v)
	if !v.Type().ConvertibleTo(tConfigPtr) {
		return reflect.Value{}, false
	}

	v = v.Convert(tConfigPtr)
	return v.Elem(), true
}

func pointerize(t, base reflect.Type, v reflect.Value) reflect.Value {
	if t == base {
		return v
	}

	if t.Kind() == reflect.Interface {
		return v
	}

	for t != v.Type() {
		if !v.CanAddr() {
			tmp := reflect.New(v.Type())
			tmp.Elem().Set(v)
			v = tmp
		} else {
			v = v.Addr()
		}
	}
	return v
}

func isInt(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true
	default:
		return false
	}
}

func isUint(k reflect.Kind) bool {
	switch k {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

func isFloat(k reflect.Kind) bool {
	switch k {
	case reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

type fieldInfo struct {
	name          string
	ftype         reflect.Type
	value         reflect.Value
	options       *options
	tagOptions    tagOptions
	validatorTags []validatorTag
}

func accessField(structVal reflect.Value, fieldIdx int, opts *options) (fieldInfo, bool, Error) {
	stField := structVal.Type().Field(fieldIdx)

	// ignore non exported fields
	if rune, _ := utf8.DecodeRuneInString(stField.Name); !unicode.IsUpper(rune) {
		return fieldInfo{}, true, nil
	}
	name, tagOpts := parseTags(stField.Tag.Get(opts.tag))
	if tagOpts.ignore {
		return fieldInfo{}, true, nil
	}

	// create new context, overwriting configValueHandling for all sub-operations
	if tagOpts.cfgHandling != opts.configValueHandling {
		tmp := &options{}
		*tmp = *opts
		tmp.configValueHandling = tagOpts.cfgHandling
		opts = tmp
	}

	validators, err := parseValidatorTags(stField.Tag.Get(opts.validatorTag))
	if err != nil {
		return fieldInfo{}, false, raiseCritical(err, "")
	}

	return fieldInfo{
		name:          fieldName(name, stField.Name),
		ftype:         stField.Type,
		value:         structVal.Field(fieldIdx),
		options:       opts,
		tagOptions:    tagOpts,
		validatorTags: validators,
	}, false, nil
}
