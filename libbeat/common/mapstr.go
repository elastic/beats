package common

import (
	"encoding/json"
	"fmt"
	"time"
)

// Commonly used map of things, used in JSON creation and the like.
type MapStr map[string]interface{}

// Eventer defines a type its ability to fill a MapStr.
type Eventer interface {
	// Add fields to MapStr.
	Event(event MapStr) error
}

// MapStrUnion creates a new MapStr containing the union of the
// key-value pairs of the two maps. If the same key is present in
// both, the key-value pairs from dict2 overwrite the ones from dict1.
func MapStrUnion(dict1 MapStr, dict2 MapStr) MapStr {
	dict := MapStr{}

	for k, v := range dict1 {
		dict[k] = v
	}

	for k, v := range dict2 {
		dict[k] = v
	}
	return dict
}

// Update copies all the key-value pairs from the
// d map overwriting any existing keys.
func (m MapStr) Update(d MapStr) {
	for k, v := range d {
		m[k] = v
	}
}

// Checks if a timestamp field exists and if it doesn't it adds
// one by using the injected now() function as a time source.
func (m MapStr) EnsureTimestampField(now func() time.Time) error {
	ts, exists := m["@timestamp"]
	if !exists {
		m["@timestamp"] = Time(now())
		return nil
	}

	_, is_common_time := ts.(Time)
	if is_common_time {
		// already perfect
		return nil
	}

	tstime, is_time := ts.(time.Time)
	if is_time {
		m["@timestamp"] = Time(tstime)
		return nil
	}

	tsstr, is_string := ts.(string)
	if is_string {
		var err error
		m["@timestamp"], err = ParseTime(tsstr)
		return err
	}
	return fmt.Errorf("Don't know how to convert %v to a Time value", ts)
}

func (m MapStr) EnsureCountField() error {
	_, exists := m["count"]
	if !exists {
		m["count"] = 1
	}
	return nil
}

// Prints the dict as a json
func (m MapStr) String() string {
	bytes, err := json.Marshal(m)
	if err != nil {
		return fmt.Sprintf("Not valid json: %v", err)
	}
	return string(bytes)
}

// UnmarshalYAML helps out with the YAML unmarshalling when the target
// variable is a MapStr. The default behavior is to unmarshal nested
// maps to map[interface{}]interface{} values, and such values can't
// be marshalled as JSON.
//
// The keys of map[interface{}]interface{} maps will be converted to
// strings with a %v format string, as will any scalar values that
// aren't already strings (i.e. numbers and boolean values).
//
// Since we want to modify the receiver it needs to be a pointer.
func (ms *MapStr) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var result map[interface{}]interface{}
	err := unmarshal(&result)
	if err != nil {
		panic(err)
	}
	*ms = cleanUpInterfaceMap(result)
	return nil
}

func cleanUpInterfaceArray(in []interface{}) []interface{} {
	result := make([]interface{}, len(in))
	for i, v := range in {
		result[i] = cleanUpMapValue(v)
	}
	return result
}

func cleanUpInterfaceMap(in map[interface{}]interface{}) MapStr {
	result := make(MapStr)
	for k, v := range in {
		result[fmt.Sprintf("%v", k)] = cleanUpMapValue(v)
	}
	return result
}

func cleanUpMapValue(v interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		return cleanUpInterfaceArray(v)
	case map[interface{}]interface{}:
		return cleanUpInterfaceMap(v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}
