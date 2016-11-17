package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// Event metadata constants. These keys are used within libbeat to identify
// metadata stored in an event.
const (
	EventMetadataKey = "_event_metadata"
	FieldsKey        = "fields"
	TagsKey          = "tags"
)

var (
	// ErrKeyNotFound indicates that the specified key was not found.
	ErrKeyNotFound = errors.New("key not found")
)

// EventMetadata contains fields and tags that can be added to an event via
// configuration.
type EventMetadata struct {
	Fields          MapStr
	FieldsUnderRoot bool `config:"fields_under_root"`
	Tags            []string
}

// MapStr is a map[string]interface{} wrapper with utility methods for common
// map operations like converting to JSON.
type MapStr map[string]interface{}

// Update copies all the key-value pairs from d to this map. If the key
// already exists then it is overwritten. This method does not merge nested
// maps.
func (m MapStr) Update(d MapStr) {
	for k, v := range d {
		m[k] = v
	}
}

// Delete deletes the given key from the map.
func (m MapStr) Delete(key string) error {
	_, err := walkMap(key, m, opDelete)
	return err
}

// CopyFieldsTo copies the field specified by key to the given map. It will
// overwrite the key if it exists. An error is returned if the key does not
// exist in the source map.
func (m MapStr) CopyFieldsTo(to MapStr, key string) error {
	v, err := walkMap(key, m, opGet)
	if err != nil {
		return err
	}

	_, err = walkMap(key, to, mapStrOperation{putOperation{v}, true})
	return err
}

// Clone returns a copy of the MapStr. It recursively makes copies of inner
// maps.
func (m MapStr) Clone() MapStr {
	result := MapStr{}

	for k, v := range m {
		innerMap, err := toMapStr(v)
		if err == nil {
			result[k] = innerMap.Clone()
		} else {
			result[k] = v
		}
	}

	return result
}

// HasKey returns true if the key exist. If an error occurs then false is
// returned with a non-nil error.
func (m MapStr) HasKey(key string) (bool, error) {
	hasKey, err := walkMap(key, m, opHasKey)
	if err != nil {
		return false, err
	}

	return hasKey.(bool), nil
}

// GetValue gets a value from the map. If the key does not exist then an error
// is returned.
func (m MapStr) GetValue(key string) (interface{}, error) {
	return walkMap(key, m, opGet)
}

// Put associates the specified value with the specified key. If the map
// previously contained a mapping for the key, the old value is replaced and
// returned. The key can be expressed in dot-notation (e.g. x.y) to put a value
// into a nested map.
//
// If you need insert keys containing dots then you must use bracket notation
// to insert values (e.g. m[key] = value).
func (m MapStr) Put(key string, value interface{}) (interface{}, error) {
	return walkMap(key, m, mapStrOperation{putOperation{value}, true})
}

// StringToPrint returns the MapStr as pretty JSON.
func (m MapStr) StringToPrint() string {
	json, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Sprintf("Not valid json: %v", err)
	}
	return string(json)
}

// String returns the MapStr as JSON.
func (m MapStr) String() string {
	bytes, err := json.Marshal(m)
	if err != nil {
		return fmt.Sprintf("Not valid json: %v", err)
	}
	return string(bytes)
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

// MergeFields merges the top-level keys and values in each source map (it does
// not perform a deep merge). If the same key exists in both, the value in
// fields takes precedence. If underRoot is true then the contents of the fields
// MapStr is merged with the value of the 'fields' key in ms.
//
// An error is returned if underRoot is true and the value of ms.fields is not a
// MapStr.
func MergeFields(ms, fields MapStr, underRoot bool) error {
	if ms == nil || fields == nil {
		return nil
	}

	fieldsMS := ms
	if !underRoot {
		f, ok := ms[FieldsKey]
		if !ok {
			fieldsMS = make(MapStr, len(fields))
			ms[FieldsKey] = fieldsMS
		} else {
			// Use existing 'fields' value.
			var err error
			fieldsMS, err = toMapStr(f)
			if err != nil {
				return err
			}
		}
	}

	// Add fields and override.
	for k, v := range fields {
		fieldsMS[k] = v
	}

	return nil
}

// AddTags appends a tag to the tags field of ms. If the tags field does not
// exist then it will be created. If the tags field exists and is not a []string
// then an error will be returned. It does not deduplicate the list of tags.
func AddTags(ms MapStr, tags []string) error {
	if ms == nil || len(tags) == 0 {
		return nil
	}

	tagsIfc, ok := ms[TagsKey]
	if !ok {
		ms[TagsKey] = tags
		return nil
	}

	existingTags, ok := tagsIfc.([]string)
	if !ok {
		return errors.Errorf("expected string array by type is %T", tagsIfc)
	}

	ms[TagsKey] = append(existingTags, tags...)
	return nil
}

// toMapStr performs a type assertion on v and returns a MapStr. v can be either
// a MapStr or a map[string]interface{}. If it's any other type or nil then
// an error is returned.
func toMapStr(v interface{}) (MapStr, error) {
	switch v.(type) {
	case MapStr:
		return v.(MapStr), nil
	case map[string]interface{}:
		m := v.(map[string]interface{})
		return MapStr(m), nil
	default:
		return nil, errors.Errorf("expected map but type is %T", v)
	}
}

// walkMap walks the data MapStr to arrive at the value specified by the key.
// The key is expressed in dot-notation (eg. x.y.z). When the key is found then
// the given mapStrOperation is invoked.
func walkMap(key string, data MapStr, op mapStrOperation) (interface{}, error) {
	var err error
	keyParts := strings.Split(key, ".")

	// Walk maps until reaching a leaf object.
	m := data
	for i, k := range keyParts[0 : len(keyParts)-1] {
		v, exists := m[k]
		if !exists {
			if op.CreateMissingKeys {
				newMap := MapStr{}
				m[k] = newMap
				m = newMap
				continue
			}
			return nil, errors.Wrapf(ErrKeyNotFound, "key=%v", strings.Join(keyParts[0:i+1], "."))
		}

		m, err = toMapStr(v)
		if err != nil {
			return nil, errors.Wrapf(err, "key=%v", strings.Join(keyParts[0:i+1], "."))
		}
	}

	// Execute the mapStrOperator on the leaf object.
	v, err := op.Do(keyParts[len(keyParts)-1], m)
	if err != nil {
		return nil, errors.Wrapf(err, "key=%v", key)
	}

	return v, nil
}

// mapStrOperation types

// These are static mapStrOperation types that store no state and are reusable.
var (
	opDelete = mapStrOperation{deleteOperation{}, false}
	opGet    = mapStrOperation{getOperation{}, false}
	opHasKey = mapStrOperation{hasKeyOperation{}, false}
)

// mapStrOperation represents an operation that can be applied to map.
type mapStrOperation struct {
	mapStrOperator
	CreateMissingKeys bool
}

// mapStrOperator is an interface with a single function that performs an
// operation on a MapStr.
type mapStrOperator interface {
	Do(key string, data MapStr) (value interface{}, err error)
}

type deleteOperation struct{}

func (op deleteOperation) Do(key string, data MapStr) (interface{}, error) {
	value, found := data[key]
	if !found {
		return nil, ErrKeyNotFound
	}
	delete(data, key)
	return value, nil
}

type getOperation struct{}

func (op getOperation) Do(key string, data MapStr) (interface{}, error) {
	value, found := data[key]
	if !found {
		return nil, ErrKeyNotFound
	}
	return value, nil
}

type hasKeyOperation struct{}

func (op hasKeyOperation) Do(key string, data MapStr) (interface{}, error) {
	_, found := data[key]
	return found, nil
}

type putOperation struct {
	Value interface{}
}

func (op putOperation) Do(key string, data MapStr) (interface{}, error) {
	existingValue, _ := data[key]
	data[key] = op.Value
	return existingValue, nil
}
