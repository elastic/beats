/*
The conversion functions take a key and a map[string]string. First it checks if the key exists and logs and error
if this is not the case. Second the conversion to the type is done. In case of an error and error is logged and the
default values is returned. This guarantees that also if a field is missing or is not defined, still the full metricset
is returned.
*/
package helper

import (
	"strconv"

	"github.com/elastic/beats/libbeat/logp"
)

// ToBool converts value to bool. In case of error, returns false
func ToBool(key string, data map[string]string) bool {

	exists := checkExist(key, data)
	if !exists {
		logp.Err("Key does not exist in in data: %s", key)
		return false
	}

	value, err := strconv.ParseBool(data[key])
	if err != nil {
		logp.Err("Error converting param to bool: %s", key)
		return false
	}

	return value
}

// ToFloat converts value to float64. In case of error, returns 0.0
func ToFloat(key string, data map[string]string) float64 {

	exists := checkExist(key, data)
	if !exists {
		logp.Err("Key does not exist in in data: %s", key)
		return 0.0
	}

	value, err := strconv.ParseFloat(data[key], 64)
	if err != nil {
		logp.Err("Error converting param to float: %s", key)
		value = 0.0
	}

	return value
}

// ToInt converts value to int. In case of error, returns 0
func ToInt(key string, data map[string]string) int64 {

	exists := checkExist(key, data)
	if !exists {
		logp.Err("Key does not exist in in data: %s", key)
		return 0
	}

	value, err := strconv.ParseInt(data[key], 10, 64)
	if err != nil {
		logp.Err("Error converting param to int: %s", key)
		return 0
	}

	return value
}

// ToStr converts value to str. In case of error, returns ""
func ToStr(key string, data map[string]string) string {

	exists := checkExist(key, data)
	if !exists {
		logp.Err("Key does not exist in in data: %s", key)
		return ""
	}

	return data[key]
}

// checkExists checks if a key exists in the given data set
func checkExist(key string, data map[string]string) bool {
	_, ok := data[key]
	return ok
}
