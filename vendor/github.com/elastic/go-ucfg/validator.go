package ucfg

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Validator interface {
	Validate() error
}

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
	RegisterValidator("nonzero", validateNonZero)
	RegisterValidator("positive", validatePositive)
	RegisterValidator("min", validateMin)
	RegisterValidator("max", validateMax)
	RegisterValidator("required", validateRequired)
}

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
	for _, tag := range validators {
		if err := tag.cb(val, tag.param); err != nil {
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

	val := reflect.ValueOf(v)
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
	return validateNonEmpty(v, name)
}

func validateNonEmpty(v interface{}, _ string) error {
	if s, ok := v.(string); ok {
		if s == "" {
			return ErrEmpty
		}
		return nil
	}

	if r, ok := v.(regexp.Regexp); ok {
		if r.String() == "" {
			return ErrEmpty
		}
		return nil
	}

	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Array || val.Kind() == reflect.Slice {
		if val.Len() == 0 {
			return ErrEmpty
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
