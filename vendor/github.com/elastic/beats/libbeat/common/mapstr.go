// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package common

import (
	"encoding/json"
	"fmt"
	"io"
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
// DeepUpdateNoOverwrite is a version of this function that does not
// overwrite existing values.
func (m MapStr) DeepUpdate(d MapStr) {
	m.deepUpdateMap(d, true)
}

// DeepUpdateNoOverwrite recursively copies the key-value pairs from d to this map.
// If a key is already present it will not be overwritten.
// DeepUpdate is a version of this function that overwrites existing values.
func (m MapStr) DeepUpdateNoOverwrite(d MapStr) {
	m.deepUpdateMap(d, false)
}

func (m MapStr) deepUpdateMap(d MapStr, overwrite bool) {
	for k, v := range d {
		switch val := v.(type) {
		case map[string]interface{}:
			m[k] = deepUpdateValue(m[k], MapStr(val), overwrite)
		case MapStr:
			m[k] = deepUpdateValue(m[k], val, overwrite)
		default:
			if overwrite {
				m[k] = v
			} else if _, exists := m[k]; !exists {
				m[k] = v
			}
		}
	}
}

func deepUpdateValue(old interface{}, val MapStr, overwrite bool) interface{} {
	if old == nil {
		return val
	}

	switch sub := old.(type) {
	case MapStr:
		sub.deepUpdateMap(val, overwrite)
		return sub
	case map[string]interface{}:
		tmp := MapStr(sub)
		tmp.deepUpdateMap(val, overwrite)
		return tmp
	default:
		// This should never happen
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

	debugM := m.Clone()
	applyLoggingMask(map[string]interface{}(debugM))

	keys := make([]string, 0, len(debugM))
	for k := range debugM {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := debugM[k]
		if inner, ok := tryToMapStr(v); ok {
			enc.AddObject(k, inner)
			continue
		}
		zap.Any(k, v).AddTo(enc)
	}
	return nil
}

// Format implements fmt.Formatter
func (m MapStr) Format(f fmt.State, c rune) {
	if f.Flag('+') || f.Flag('#') {
		io.WriteString(f, m.String())
		return
	}

	debugM := m.Clone()
	applyLoggingMask(map[string]interface{}(debugM))

	io.WriteString(f, debugM.String())
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
// MapStr is merged with the value of the 'fields' key in target.
//
// An error is returned if underRoot is true and the value of ms.fields is not a
// MapStr.
func MergeFields(target, from MapStr, underRoot bool) error {
	if target == nil || len(from) == 0 {
		return nil
	}

	destMap, err := mergeFieldsGetDestMap(target, from, underRoot)
	if err != nil {
		return err
	}

	// Add fields and override.
	for k, v := range from {
		destMap[k] = v
	}

	return nil
}

// MergeFieldsDeep recursively merges the keys and values from `from` into `target`, either
// into ms itself (if underRoot == true) or into ms["fields"] (if underRoot == false). If
// the same key exists in `from` and the destination map, the value in fields takes precedence.
//
// An error is returned if underRoot is true and the value of ms["fields"] is not a
// MapStr.
func MergeFieldsDeep(target, from MapStr, underRoot bool) error {
	if target == nil || len(from) == 0 {
		return nil
	}

	destMap, err := mergeFieldsGetDestMap(target, from, underRoot)
	if err != nil {
		return err
	}

	destMap.DeepUpdate(from)
	return nil
}

func mergeFieldsGetDestMap(target, from MapStr, underRoot bool) (MapStr, error) {
	destMap := target
	if !underRoot {
		f, ok := target[FieldsKey]
		if !ok {
			destMap = make(MapStr, len(from))
			target[FieldsKey] = destMap
		} else {
			// Use existing 'fields' value.
			var err error
			destMap, err = toMapStr(f)
			if err != nil {
				return nil, err
			}
		}
	}

	return destMap, nil
}

// AddTags appends a tag to the tags field of ms. If the tags field does not
// exist then it will be created. If the tags field exists and is not a []string
// then an error will be returned. It does not deduplicate the list of tags.
func AddTags(ms MapStr, tags []string) error {
	return AddTagsWithKey(ms, TagsKey, tags)
}

// AddTagsWithKey appends a tag to the key field of ms. If the field does not
// exist then it will be created. If the field exists and is not a []string
// then an error will be returned. It does not deduplicate the list.
func AddTagsWithKey(ms MapStr, key string, tags []string) error {
	if ms == nil || len(tags) == 0 {
		return nil
	}

	k, subMap, oldTags, present, err := mapFind(key, ms, true)
	if err != nil {
		return err
	}

	if !present {
		subMap[k] = tags
		return nil
	}

	switch arr := oldTags.(type) {
	case []string:
		subMap[k] = append(arr, tags...)
	case []interface{}:
		for _, tag := range tags {
			arr = append(arr, tag)
		}
		subMap[k] = arr
	default:
		return errors.Errorf("expected string array by type is %T", oldTags)

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
