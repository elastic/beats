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
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Validator interface provides additional validation support to Unpack. The
// Validate method will be executed for any type passed directly or indirectly to
// Unpack.
//
// If Validate fails with an error message, Unpack will add some
// context - like setting being accessed and file setting was read from - to the
// error message before returning the actual error.
type Validator interface {
	Validate() error
}

// ValidatorCallback is the type of optional validator tags to be registered via
// RegisterValidator.
type ValidatorCallback func(interface{}, string) error

type validatorTag struct {
	name  string
	cb    ValidatorCallback
	param string
}

var (
	validators = map[string]ValidatorCallback{}
)

func init() {
	initRegisterValidator("nonzero", validateNonZero)
	initRegisterValidator("positive", validatePositive)
	initRegisterValidator("min", validateMin)
	initRegisterValidator("max", validateMax)
	initRegisterValidator("required", validateRequired)
}

func initRegisterValidator(name string, cb ValidatorCallback) {
	if err := RegisterValidator(name, cb); err != nil {
		panic("Duplicate validator: " + name)
	}
}

// RegisterValidator adds a new validator option to the "validate" struct tag.
// The callback will be executed when unpacking into a struct field.
func RegisterValidator(name string, cb ValidatorCallback) error {
	if _, exists := validators[name]; exists {
		return ErrDuplicateValidator
	}

	validators[name] = cb
	return nil
}

func parseValidatorTags(tag string) ([]validatorTag, error) {
	if tag == "" {
		return nil, nil
	}

	lst := strings.Split(tag, ",")
	if len(lst) == 0 {
		return nil, nil
	}

	tags := make([]validatorTag, 0, len(lst))
	for _, cfg := range lst {
		v := strings.SplitN(cfg, "=", 2)
		name := strings.Trim(v[0], " \t\r\n")
		cb := validators[name]
		if cb == nil {
			return nil, fmt.Errorf("unknown validator '%v'", name)
		}

		param := ""
		if len(v) == 2 {
			param = strings.Trim(v[1], " \t\r\n")
		}

		tags = append(tags, validatorTag{name: name, cb: cb, param: param})
	}

	return tags, nil
}

func tryValidate(val reflect.Value) error {
	t := val.Type()
	var validator Validator

	if (t.Kind() == reflect.Ptr || t.Kind() == reflect.Interface) && val.IsNil() {
		return nil
	}

	if t.Implements(tValidator) {
		validator = val.Interface().(Validator)
	} else if reflect.PtrTo(t).Implements(tValidator) {
		val = pointerize(reflect.PtrTo(t), t, val)
		validator = val.Interface().(Validator)
	}

	if validator == nil {
		return nil
	}
	return validator.Validate()
}

func runValidators(val interface{}, validators []validatorTag) error {
	if validators == nil {
		return nil
	}
	for _, tag := range validators {
		if err := tag.cb(val, tag.param); err != nil {
			return err
		}
	}
	return nil
}

func tryRecursiveValidate(val reflect.Value, opts *options, validators []validatorTag) error {
	var curr interface{}
	if val.IsValid() {
		curr = val.Interface()
	}
	if err := runValidators(curr, validators); err != nil {
		return err
	}
	if !val.IsValid() {
		return nil
	}

	t := val.Type()
	if (t.Kind() == reflect.Ptr || t.Kind() == reflect.Interface) && val.IsNil() {
		return nil
	}

	var err error
	switch chaseValue(val).Kind() {
	case reflect.Struct:
		err = validateStruct(val, opts)
	case reflect.Map:
		err = validateMap(val, opts)
	case reflect.Array, reflect.Slice:
		err = validateArray(val, opts)
	}

	if err != nil {
		return err
	}
	return tryValidate(val)
}

func validateStruct(val reflect.Value, opts *options) error {
	val = chaseValue(val)
	numField := val.NumField()
	for i := 0; i < numField; i++ {
		fInfo, skip, err := accessField(val, i, opts)
		if err != nil {
			return err
		}
		if skip {
			continue
		}

		if err := tryRecursiveValidate(fInfo.value, fInfo.options, fInfo.validatorTags); err != nil {
			return err
		}
	}
	return nil
}

func validateMap(val reflect.Value, opts *options) error {
	for _, key := range val.MapKeys() {
		if err := tryRecursiveValidate(val.MapIndex(key), opts, nil); err != nil {
			return err
		}
	}
	return nil
}

func validateArray(val reflect.Value, opts *options) error {
	for i := 0; i < val.Len(); i++ {
		if err := tryRecursiveValidate(val.Index(i), opts, nil); err != nil {
			return err
		}
	}
	return nil
}

// validateNonZero implements the `nonzero` validation tag.
// If nonzero is set, the validator is only run if field is present in config.
// It checks for numbers and durations to be != 0, and for strings/arrays/slices
// not being empty.
func validateNonZero(v interface{}, name string) error {
	if v == nil {
		return nil
	}

	if d, ok := v.(time.Duration); ok {
		if d == 0 {
			return ErrZeroValue
		}
		return nil
	}

	val := chaseValue(reflect.ValueOf(v))
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val.Int() != 0 {
			return nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val.Uint() != 0 {
			return nil
		}
	case reflect.Float32, reflect.Float64:
		if val.Float() != 0 {
			return nil
		}
	default:
		return validateNonEmpty(v, name)
	}

	return ErrZeroValue
}

func validatePositive(v interface{}, _ string) error {
	if v == nil {
		return nil
	}

	if d, ok := v.(time.Duration); ok {
		if d < 0 {
			return ErrNegative
		}
		return nil
	}

	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val.Int() >= 0 {
			return nil
		}
	case reflect.Float32, reflect.Float64:
		if val.Float() >= 0 {
			return nil
		}
	default:
		return nil
	}

	return ErrNegative
}

func validateMin(v interface{}, param string) error {
	if v == nil {
		return nil
	}

	if d, ok := v.(time.Duration); ok {
		min, err := param2Duration(param)
		if err != nil {
			return err
		}

		if min > d {
			return fmt.Errorf("requires duration < %v", param)
		}
		return nil
	}

	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		min, err := strconv.ParseInt(param, 0, 64)
		if err != nil {
			return err
		}
		if val.Int() >= min {
			return nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		min, err := strconv.ParseUint(param, 0, 64)
		if err != nil {
			return err
		}
		if val.Uint() >= min {
			return nil
		}
	case reflect.Float32, reflect.Float64:
		min, err := strconv.ParseFloat(param, 64)
		if err != nil {
			return err
		}
		if val.Float() >= min {
			return nil
		}
	default:
		return nil
	}

	return fmt.Errorf("requires value < %v", param)
}

func validateMax(v interface{}, param string) error {
	if v == nil {
		return nil
	}

	if d, ok := v.(time.Duration); ok {
		max, err := param2Duration(param)
		if err != nil {
			return err
		}

		if max < d {
			return fmt.Errorf("requires duration > %v", param)
		}
		return nil
	}

	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		max, err := strconv.ParseInt(param, 0, 64)
		if err != nil {
			return err
		}
		if val.Int() <= max {
			return nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		max, err := strconv.ParseUint(param, 0, 64)
		if err != nil {
			return err
		}
		if val.Uint() <= max {
			return nil
		}
	case reflect.Float32, reflect.Float64:
		max, err := strconv.ParseFloat(param, 64)
		if err != nil {
			return err
		}
		if val.Float() <= max {
			return nil
		}
	default:
		return nil
	}

	return fmt.Errorf("requires value > %v", param)
}

// validateRequired implements the `required` validation tag.
// If a field is required, it must be present in the config.
// If field is a string, regex or slice its length must be > 0.
func validateRequired(v interface{}, name string) error {
	if v == nil {
		return ErrRequired
	}
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr && val.IsNil() {
		return ErrRequired
	}
	if isInt(val.Kind()) || isUint(val.Kind()) || isFloat(val.Kind()) {
		if err := validateNonZero(v, name); err != nil {
			return ErrRequired
		}
		return nil
	}
	if err := validateNonEmptyWithAllowNil(v, name, false); err != nil {
		return err
	}
	return nil
}

func validateNonEmpty(v interface{}, name string) error {
	return validateNonEmptyWithAllowNil(v, name, true)
}

func validateNonEmptyWithAllowNil(v interface{}, _ string, allowNil bool) error {
	if s, ok := v.(string); ok {
		if s == "" {
			return ErrStringEmpty
		}
		return nil
	}

	if r, ok := v.(regexp.Regexp); ok {
		if r.String() == "" {
			return ErrRegexEmpty
		}
		return nil
	}

	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Array || val.Kind() == reflect.Slice {
		if val.IsNil() {
			if allowNil {
				return nil
			}
			return ErrRequired
		}
		if val.Len() == 0 {
			return ErrArrayEmpty
		}
		return nil
	}
	if val.Kind() == reflect.Map {
		if val.IsNil() {
			if allowNil {
				return nil
			}
			return ErrRequired
		}
		if val.Len() == 0 {
			return ErrMapEmpty
		}
		return nil
	}

	return nil
}

func param2Duration(param string) (time.Duration, error) {
	d, err := time.ParseDuration(param)
	if err == nil {
		return d, err
	}

	tmp, floatErr := strconv.ParseFloat(param, 64)
	if floatErr != nil {
		return 0, err
	}

	return time.Duration(tmp * float64(time.Second)), nil
}
