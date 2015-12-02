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
