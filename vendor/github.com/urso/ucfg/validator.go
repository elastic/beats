package ucfg

import (
	"fmt"
	"reflect"
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

func tryValidate(val interface{}) error {
	if v, ok := val.(Validator); ok {
		return v.Validate()
	}
	return nil
}

func runValidators(val interface{}, validators []validatorTag) error {
	for _, tag := range validators {
		if err := tag.cb(val, tag.param); err != nil {
			return err
		}
	}
	return nil
}

func validateNonZero(v interface{}, _ string) error {
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
		return nil
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

func validateRequired(v interface{}, _ string) error {
	if v != nil {
		return nil
	}
	return ErrRequired
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
