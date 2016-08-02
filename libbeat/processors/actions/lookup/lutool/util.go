package lutool

import (
	"errors"
	"reflect"
	"strconv"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func makeFieldsCollector(fields []string) (func(common.MapStr) ([]string, bool), error) {
	type stringer interface {
		String() string
	}

	b := func(event common.MapStr) ([]string, bool) {
		keys := make([]string, len(fields))
		return collectFieldsInto(keys, fields, event)
	}

	if len(fields) == 0 {
		return nil, errors.New("No keys configured")
	}
	return b, nil
}

func collectFieldsInto(
	to []string,
	fields []string,
	event common.MapStr,
) ([]string, bool) {

	if len(to) != len(fields) {
		return nil, false
	}

	for i, f := range fields {
		s, err := fieldString(event, f)
		if err != nil {
			return nil, false
		}

		to[i] = s
	}

	return to, true
}

// TODO: move to libbeat/common and remove duplicate code
func fieldString(event common.MapStr, field string) (string, error) {
	type stringer interface {
		String() string
	}

	v, err := event.GetValue(field)
	if err != nil {
		return "", err
	}

	switch s := v.(type) {
	case string:
		return s, nil
	case []byte:
		return string(s), nil
	case stringer:
		return s.String(), nil
	case int8, int16, int32, int64, int:
		i := reflect.ValueOf(s).Int()
		return strconv.FormatInt(i, 16), nil
	case uint8, uint16, uint32, uint64, uint:
		u := reflect.ValueOf(s).Uint()
		return strconv.FormatUint(u, 16), nil
	case float32:
		return strconv.FormatFloat(float64(s), 'g', -1, 32), nil
	case float64:
		return strconv.FormatFloat(s, 'g', -1, 64), nil
	default:
		logp.Warn("Can not convert key '%v' value to string", v)
		return "", errConvertString
	}
}
