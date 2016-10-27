package common

import (
	"encoding/json"
)

// transformNumbersDict walks a json decoded tree an replaces json.Number
// with int64, float64, or string, in this order of preference (i.e. if it
// parses as an int, use int. if it parses as a float, use float. etc).
func TransformNumbersDict(dict MapStr) {
	for k, v := range dict {
		switch vv := v.(type) {
		case json.Number:
			dict[k] = TransformNumber(vv)
		case map[string]interface{}:
			TransformNumbersDict(vv)
		case []interface{}:
			TransformNumbersArray(vv)
		}
	}
}

func TransformNumber(value json.Number) interface{} {
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

func TransformNumbersArray(arr []interface{}) {
	for i, v := range arr {
		switch vv := v.(type) {
		case json.Number:
			arr[i] = TransformNumber(vv)
		case map[string]interface{}:
			TransformNumbersDict(vv)
		case []interface{}:
			TransformNumbersArray(vv)
		}
	}
}
