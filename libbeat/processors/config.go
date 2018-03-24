package processors

import (
	"fmt"
	"math"
	"strconv"

	"github.com/elastic/beats/libbeat/common"
)

type ConditionConfig struct {
	Equals    *ConditionFields  `config:"equals"`
	Contains  *ConditionFields  `config:"contains"`
	Regexp    *ConditionFields  `config:"regexp"`
	Range     *ConditionFields  `config:"range"`
	HasFields []string          `config:"has_fields"`
	OR        []ConditionConfig `config:"or"`
	AND       []ConditionConfig `config:"and"`
	NOT       *ConditionConfig  `config:"not"`
}

type ConditionFields struct {
	fields map[string]interface{}
}

type PluginConfig []map[string]*common.Config

// fields that should be always exported
var MandatoryExportedFields = []string{"type"}

func (f *ConditionFields) Unpack(to interface{}) error {
	m, ok := to.(map[string]interface{})
	if !ok {
		return fmt.Errorf("wrong type, expect map")
	}

	f.fields = map[string]interface{}{}

	var expand func(key string, value interface{})

	expand = func(key string, value interface{}) {
		switch v := value.(type) {
		case map[string]interface{}:
			for k, val := range v {
				expand(fmt.Sprintf("%v.%v", key, k), val)
			}
		case []interface{}:
			for i := range v {
				expand(fmt.Sprintf("%v.%v", key, i), v[i])
			}
		default:
			f.fields[key] = value
		}
	}

	for k, val := range m {
		expand(k, val)
	}
	return nil
}

func extractFloat(unk interface{}) (float64, error) {
	switch i := unk.(type) {
	case float64:
		return float64(i), nil
	case float32:
		return float64(i), nil
	case int64:
		return float64(i), nil
	case int32:
		return float64(i), nil
	case int16:
		return float64(i), nil
	case int8:
		return float64(i), nil
	case uint64:
		return float64(i), nil
	case uint32:
		return float64(i), nil
	case uint16:
		return float64(i), nil
	case uint8:
		return float64(i), nil
	case int:
		return float64(i), nil
	case uint:
		return float64(i), nil
	case string:
		f, err := strconv.ParseFloat(i, 64)
		if err != nil {
			return math.NaN(), err
		}
		return f, err
	default:
		return math.NaN(), fmt.Errorf("unknown type %T passed to extractFloat", unk)
	}
}

func extractInt(unk interface{}) (uint64, error) {
	switch i := unk.(type) {
	case int64:
		return uint64(i), nil
	case int32:
		return uint64(i), nil
	case int16:
		return uint64(i), nil
	case int8:
		return uint64(i), nil
	case uint64:
		return uint64(i), nil
	case uint32:
		return uint64(i), nil
	case uint16:
		return uint64(i), nil
	case uint8:
		return uint64(i), nil
	case int:
		return uint64(i), nil
	case uint:
		return uint64(i), nil
	default:
		return 0, fmt.Errorf("unknown type %T passed to extractInt", unk)
	}
}

func extractString(unk interface{}) (string, error) {
	switch s := unk.(type) {
	case string:
		return s, nil
	default:
		return "", fmt.Errorf("unknown type %T passed to extractString", unk)
	}
}

func extractBool(unk interface{}) (bool, error) {
	switch b := unk.(type) {
	case bool:
		return b, nil
	default:
		return false, fmt.Errorf("unknown type %T passed to extractBool", unk)
	}
}
