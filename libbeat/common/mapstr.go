package common

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Event metadata constants. These keys are used within libbeat to identify
// metadata stored in an event.
const (
	FieldsKey = "fields"
	TagsKey   = "tags"
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

// DeepUpdate recursively copies the key-value pairs from d to this map.
// If the key is present and a map as well, the sub-map will be updated recursively
// via DeepUpdate.
func (m MapStr) DeepUpdate(d MapStr) {
	for k, v := range d {
		switch val := v.(type) {
		case map[string]interface{}:
			m[k] = deepUpdateValue(m[k], MapStr(val))
		case MapStr:
			m[k] = deepUpdateValue(m[k], val)
		default:
			m[k] = v
		}
	}
}

func deepUpdateValue(old interface{}, val MapStr) interface{} {
	if old == nil {
		return val
	}

	switch sub := old.(type) {
	case MapStr:
		sub.DeepUpdate(val)
		return sub
	case map[string]interface{}:
		tmp := MapStr(sub)
		tmp.DeepUpdate(val)
		return tmp
	default:
		return val
	}
}

// Delete deletes the given key from the map.
func (m MapStr) Delete(key string) error {
	k, d, _, found, err := mapFind(key, m, false)
	if err != nil {
		return err
	}
	if !found {
		return ErrKeyNotFound
	}

	delete(d, k)
	return nil
}

// CopyFieldsTo copies the field specified by key to the given map. It will
// overwrite the key if it exists. An error is returned if the key does not
// exist in the source map.
func (m MapStr) CopyFieldsTo(to MapStr, key string) error {
	v, err := m.GetValue(key)
	if err != nil {
		return err
	}

	_, err = to.Put(key, v)
	return err
}

// Clone returns a copy of the MapStr. It recursively makes copies of inner
// maps.
func (m MapStr) Clone() MapStr {
	result := MapStr{}

	for k, v := range m {
		if innerMap, ok := tryToMapStr(v); ok {
			v = innerMap.Clone()
		}
		result[k] = v
	}

	return result
}

// HasKey returns true if the key exist. If an error occurs then false is
// returned with a non-nil error.
func (m MapStr) HasKey(key string) (bool, error) {
	_, _, _, hasKey, err := mapFind(key, m, false)
	return hasKey, err
}

// GetValue gets a value from the map. If the key does not exist then an error
// is returned.
func (m MapStr) GetValue(key string) (interface{}, error) {
	_, _, v, found, err := mapFind(key, m, false)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrKeyNotFound
	}
	return v, nil
}

// Put associates the specified value with the specified key. If the map
// previously contained a mapping for the key, the old value is replaced and
// returned. The key can be expressed in dot-notation (e.g. x.y) to put a value
// into a nested map.
//
// If you need insert keys containing dots then you must use bracket notation
// to insert values (e.g. m[key] = value).
func (m MapStr) Put(key string, value interface{}) (interface{}, error) {
	// XXX `safemapstr.Put` mimics this implementation, both should be updated to have similar behavior
	k, d, old, _, err := mapFind(key, m, true)
	if err != nil {
		return nil, err
	}

	d[k] = value
	return old, nil
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

// MarshalLogObject implements the zapcore.ObjectMarshaler interface and allows
// for more efficient marshaling of MapStr in structured logging.
func (m MapStr) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if len(m) == 0 {
		return nil
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := m[k]
		if inner, ok := tryToMapStr(v); ok {
			enc.AddObject(k, inner)
			continue
		}
		zap.Any(k, v).AddTo(enc)
	}
	return nil
}

// Flatten flattens the given MapStr and returns a flat MapStr.
//
// Example:
//   "hello": MapStr{"world": "test" }
//
// This is converted to:
//   "hello.world": "test"
//
// This can be useful for testing or logging.
func (m MapStr) Flatten() MapStr {
	return flatten("", m, MapStr{})
}

// flatten is a helper for Flatten. See docs for Flatten. For convenience the
// out parameter is returned.
func flatten(prefix string, in, out MapStr) MapStr {
	for k, v := range in {
		var fullKey string
		if prefix == "" {
			fullKey = k
		} else {
			fullKey = fmt.Sprintf("%s.%s", prefix, k)
		}

		if m, ok := tryToMapStr(v); ok {
			flatten(fullKey, m, out)
		} else {
			out[fullKey] = v
		}
	}
	return out
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
	if ms == nil || len(fields) == 0 {
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
	eventTags, exists := ms[TagsKey]
	if !exists {
		ms[TagsKey] = tags
		return nil
	}

	switch arr := eventTags.(type) {
	case []string:
		ms[TagsKey] = append(arr, tags...)
	case []interface{}:
		for _, tag := range tags {
			arr = append(arr, tag)
		}
		ms[TagsKey] = arr
	default:
		return errors.Errorf("expected string array by type is %T", eventTags)
	}
	return nil
}

// toMapStr performs a type assertion on v and returns a MapStr. v can be either
// a MapStr or a map[string]interface{}. If it's any other type or nil then
// an error is returned.
func toMapStr(v interface{}) (MapStr, error) {
	m, ok := tryToMapStr(v)
	if !ok {
		return nil, errors.Errorf("expected map but type is %T", v)
	}
	return m, nil
}

func tryToMapStr(v interface{}) (MapStr, bool) {
	switch m := v.(type) {
	case MapStr:
		return m, true
	case map[string]interface{}:
		return MapStr(m), true
	default:
		return nil, false
	}
}

// mapFind iterates a MapStr based on a the given dotted key, finding the final
// subMap and subKey to operate on.
// An error is returned if some intermediate is no map or the key doesn't exist.
// If createMissing is set to true, intermediate maps are created.
// The final map and un-dotted key to run further operations on are returned in
// subKey and subMap. The subMap already contains a value for subKey, the
// present flag is set to true and the oldValue return will hold
// the original value.
func mapFind(
	key string,
	data MapStr,
	createMissing bool,
) (subKey string, subMap MapStr, oldValue interface{}, present bool, err error) {
	// XXX `safemapstr.mapFind` mimics this implementation, both should be updated to have similar behavior

	for {
		// Fast path, key is present as is.
		if v, exists := data[key]; exists {
			return key, data, v, true, nil
		}

		idx := strings.IndexRune(key, '.')
		if idx < 0 {
			return key, data, nil, false, nil
		}

		k := key[:idx]
		d, exists := data[k]
		if !exists {
			if createMissing {
				d = MapStr{}
				data[k] = d
			} else {
				return "", nil, nil, false, ErrKeyNotFound
			}
		}

		v, err := toMapStr(d)
		if err != nil {
			return "", nil, nil, false, err
		}

		// advance to sub-map
		key = key[idx+1:]
		data = v
	}
}
