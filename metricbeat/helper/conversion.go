/*
The conversion functions take a key and a map[string]string. First it checks if the key exists and logs and error
if this is not the case. Second the conversion to the type is done. In case of an error and error is logged and the
default values is returned. This guarantees that also if a field is missing or is not defined, still the full metricset
is returned.
*/
package helper

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// ToBool converts value to bool. In case of error, returns false
func ToBool(key string, data map[string]string, errs map[string]error, path string) bool {

	if !checkExist(key, data) {
		errs[path] = fmt.Errorf("Key %s not found", key)
		return false
	}

	value, err := strconv.ParseBool(data[key])
	if err != nil {
		errs[path] = fmt.Errorf("Error converting value to bool: '%s'", data[key])
		return false
	}

	return value
}

// ToFloat converts value to float64. In case of error, returns 0.0
func ToFloat(key string, data map[string]string, errs map[string]error, path string) float64 {

	if !checkExist(key, data) {
		errs[path] = fmt.Errorf("Key %s not found", key)
		return 0.0
	}

	value, err := strconv.ParseFloat(data[key], 64)
	if err != nil {
		errs[path] = fmt.Errorf("Error converting value to float: '%s'", data[key])
		value = 0.0
	}

	return value
}

// ToInt converts value to int. In case of error, returns 0
func ToInt(key string, data map[string]string, errs map[string]error, path string) int64 {

	if !checkExist(key, data) {
		errs[path] = fmt.Errorf("Key %s not found", key)
		return 0
	}

	value, err := strconv.ParseInt(data[key], 10, 64)
	if err != nil {
		errs[path] = fmt.Errorf("Error converting value to int: '%s'", data[key])
		return 0
	}

	return value
}

// ToStr converts value to str. In case of error, returns ""
func ToStr(key string, data map[string]string, errs map[string]error, path string) string {

	if !checkExist(key, data) {
		errs[path] = fmt.Errorf("Key %s not found", key)
		return ""
	}

	return data[key]
}

func RemoveErroredKeys(event common.MapStr, errs map[string]error) {
	for key, err := range errs {
		logp.Err("Error on field `%s`: %v", key, err)
		if err_ := deleteKey(event, key); err_ != nil {
			logp.Err("Error when trying to remove errored key %s: %v", key, err_)
		}
	}
}

// checkExists checks if a key exists in the given data set
func checkExist(key string, data map[string]string) bool {
	_, ok := data[key]
	return ok
}

func deleteKey(event common.MapStr, key string) error {
	path := strings.Split(key, ".")
	ev := event
	for i, pathEl := range path {
		if i == len(path)-1 {
			delete(ev, pathEl)
		} else {
			var ok bool
			ev, ok = ev[pathEl].(common.MapStr)
			if !ok {
				return fmt.Errorf("Error accessing field %s", pathEl)
			}
		}
	}
	return nil
}
