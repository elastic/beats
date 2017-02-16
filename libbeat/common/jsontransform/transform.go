package jsontransform

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
)

// TransformNumbers walks a json decoded tree an replaces json.Number
// with int64, float64, or string, in this order of preference (i.e. if it
// parses as an int, use int. if it parses as a float, use float. etc).
func TransformNumbers(dict common.MapStr) {
	for k, v := range dict {
		switch vv := v.(type) {
		case json.Number:
			dict[k] = transformNumber(vv)
		case map[string]interface{}:
			TransformNumbers(vv)
		case []interface{}:
			transformNumbersArray(vv)
		}
	}
}

func transformNumber(value json.Number) interface{} {
	i64, err := value.Int64()
	if err == nil {
		return i64
	}
	f64, err := value.Float64()
	if err == nil {
		return f64
	}
	return value.String()
}

func transformNumbersArray(arr []interface{}) {
	for i, v := range arr {
		switch vv := v.(type) {
		case json.Number:
			arr[i] = transformNumber(vv)
		case map[string]interface{}:
			TransformNumbers(vv)
		case []interface{}:
			transformNumbersArray(vv)
		}
	}
}
