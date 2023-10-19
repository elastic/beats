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

package beat

import (
	"errors"
	"fmt"
	"strings"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

type checkoutMode bool

var (
	checkoutModeOnlyMaps      checkoutMode = false
	checkoutModeIncludeValues checkoutMode = true
)

type EventError struct {
	Message   string
	Field     string
	Data      string
	Processor string
}

func (e EventError) Error() string {
	var prefix string
	if e.Processor != "" {
		prefix += fmt.Sprintf("[processor=%s] ", e.Processor)
	}

	if e.Field != "" {
		prefix += fmt.Sprintf("[field=%q] ", e.Field)
	}

	if e.Data != "" {
		prefix += fmt.Sprintf("[data=%s] ", e.Data)
	}
	return prefix + e.Message
}

func (e EventError) toMap() mapstr.M {
	m := mapstr.M{
		"message": e.Message,
	}
	if e.Data != "" {
		m["data"] = e.Data
	}
	if e.Field != "" {
		m["field"] = e.Field
	}
	if e.Processor != "" {
		m["processor"] = e.Processor
	}

	return m
}

// EventEditor is a wrapper that allows to make changes to the wrapped event
// preserving states of the original nested maps by cloning them on demand.
//
// The first time a nested map gets updated it's cloned and, from that moment on, only the copy is modified.
// Once all the changes are collected, users can call `Apply` to apply pending changes to the original event.
// When the changes get applied the pointers to originally referenced nested maps get replaced with pointers to
// modified copies.
//
// This allows us to:
// * avoid cloning the entire event and be more efficient in memory management
// * collect multiple changes and apply them at once, using a transaction-like mechanism (`Apply`/`Reset` functions).
//
// WARNING:
// Events can contains slices which this editor does not preserve.
// It's on the consumer to copy slices if they need to be changed.
// This editor takes care of maps only, not looking into slices. This means maps in slices are not preserved either.
type EventEditor struct {
	original  *Event
	pending   *Event
	deletions map[string]struct{}
}

// NewEventEditor creates a new event editor for the given event.
func NewEventEditor(e *Event) *EventEditor {
	if e == nil {
		e = &Event{}
	}
	return &EventEditor{
		original: e,
	}
}

// Fields returns the current computed state of changes without applying any changes to the original event.
//
// This function is cloning all the fields from the original event.
// Using this function is slow and expensive on memory, try not to use it unless it's absolutely necessary.
func (e *EventEditor) Fields() mapstr.M {
	if e.original.Fields == nil {
		if e.pending.Fields != nil {
			return e.pending.Fields.Clone()
		}
		return mapstr.M{}
	}
	fields := e.original.Fields.Clone()
	for key := range e.deletions {
		_ = fields.Delete(key)
	}
	if e.pending != nil {
		fields.DeepUpdate(e.pending.Fields)
	}
	return fields
}

// FlattenKeys returns all flatten keys for the current pending state of the event.
// This includes original undeleted keys and new unapplied keys.
func (e *EventEditor) FlattenKeys() []string {
	uniqueKeys := make(map[string]struct{})

	if e.original.Meta != nil {
		for _, key := range *e.original.Meta.FlattenKeys() {
			if e.checkDeleted(key) {
				continue
			}
			uniqueKeys[metadataKeyPrefix+key] = struct{}{}
		}
	}
	if e.original.Fields != nil {
		for _, key := range *e.original.Fields.FlattenKeys() {
			if e.checkDeleted(key) {
				continue
			}
			uniqueKeys[key] = struct{}{}
		}
	}

	if e.pending != nil {
		if e.pending.Meta != nil {
			for _, key := range *e.pending.Meta.FlattenKeys() {
				uniqueKeys[metadataKeyPrefix+key] = struct{}{}
			}
		}
		if e.pending.Fields != nil {
			for _, key := range *e.pending.Fields.FlattenKeys() {
				uniqueKeys[key] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(uniqueKeys))
	for key := range uniqueKeys {
		result = append(result, key)
	}
	return result
}

// GetValue gets a value from the event. The key can be expressed in dot-notation (e.g. x.y).
// If the key does not exist then `mapstr.ErrKeyNotFound` error is returned.
//
// If the returned value is a nested map this function always returns a copy,
// not the same nested map referenced by the original event.
func (e *EventEditor) GetValue(key string) (interface{}, error) {
	if key == "" {
		return nil, mapstr.ErrKeyNotFound
	}
	if key == MetadataFieldKey {
		return nil, ErrMetadataAccess
	}

	rootKey := e.rootKey(key)

	if e.pending != nil {
		// if the root key is empty the original event never had this key
		if rootKey == "" {
			return e.pending.GetValue(key)
		}
		// We try to retrieve from the `pending` event first,
		// since it should have the most recent data.
		//
		// To check if the nested map was checked-out before
		// we need to get the root-level first.
		val, err := e.pending.GetValue(rootKey)
		// if the key was found, it was either created by PutValue or
		// checked out from the original event.
		if err == nil {
			if rootKey == key {
				// the value might be the root-level end value we're looking for
				return val, nil
			} else {
				// otherwise, we need to retrieve from the nested map
				subKey := key[len(rootKey)+1:]
				switch nested := val.(type) {
				case mapstr.M:
					return nested.GetValue(subKey)
				case map[string]interface{}:
					return mapstr.M(nested).GetValue(subKey)
				default:
					return nil, mapstr.ErrKeyNotFound
				}
			}
		}

		if !errors.Is(err, mapstr.ErrKeyNotFound) {
			return nil, err
		}
	}

	// handle the deletion marks but only on the root-level
	// the rest should be covered by looking into the pending event
	// where we check out all the nested maps.
	if rootKey == key && e.checkDeleted(key) {
		return nil, mapstr.ErrKeyNotFound
	}

	// in case the key was never checked out and it's still
	// present only in the original event
	value, err := e.original.GetValue(key)
	if err != nil {
		return nil, err
	}

	// if the end value is not a map, we can just return it,
	// value types will be copied automatically
	switch value.(type) {
	case mapstr.M, map[string]interface{}:
	default:
		return value, nil
	}

	// if the key leads to a map value or it's in a nested map,
	// we must check it out before returning, so we return a clone
	// that the consumer can modify
	e.checkout(key, checkoutModeOnlyMaps)

	return e.pending.GetValue(key)
}

// PutValue associates the specified value with the specified key. If the event
// previously contained a mapping for the key, the old value is replaced and
// returned. The key can be expressed in dot-notation (e.g. x.y) to put a value
// into a nested map.
//
// If the returned previous value is a nested map this function always returns a copy,
// not the same nested map referenced by the original event.
//
// This changed is not applied to the original event until `Apply` is called.
func (e *EventEditor) PutValue(key string, v interface{}) (interface{}, error) {
	switch key {
	case MetadataFieldKey:
		return nil, ErrAlterMetadataKey
	case TimestampFieldKey:
		e.allocatePending()
		return e.pending.PutValue(key, v)
	default:
		e.allocatePending()
		// checkout only if the key leads to a nested value
		if !e.checkDeleted(key) {
			e.checkout(key, checkoutModeIncludeValues)
		}

		return e.pending.PutValue(key, v)
	}
}

// Delete value with the given key.
// The key can be expressed in dot-notation (e.g. x.y)
//
// This changed is not applied to the original event until `Apply` is called.
func (e *EventEditor) Delete(key string) error {
	if key == TimestampFieldKey {
		return ErrDeleteTimestamp
	}
	if key == MetadataFieldKey {
		return ErrAlterMetadataKey
	}
	var deleted bool
	has, _ := e.original.HasKey(key)
	if has {
		rootKey := e.rootKey(key)
		if key == rootKey {
			// if we're trying to delete a root-level key
			// it must be deleted from the `original` event when `Apply` is called
			// and from the `pending` event (below) if a value with the same key was put there.
			e.markAsDeleted(rootKey)
			deleted = true
		} else {
			// if it's not a root-level key, we checkout this entire root-level map
			// and delete from the copy in `pending` event instead.
			e.checkout(rootKey, checkoutModeOnlyMaps)
		}
	}

	if e.pending != nil {
		err := e.pending.Delete(key)
		deleted = deleted || err == nil
	}

	if deleted {
		return nil
	} else {
		return mapstr.ErrKeyNotFound
	}
}

// DeleteAll marks all data to be deleted from the event when `Apply` is called.
func (e *EventEditor) DeleteAll() {
	// reset all the changes because
	// they don't make sense anymore
	e.Reset()
	// now we mark all root level keys for deletion
	// in both `Meta` and `Fields` maps of the original event
	if e.original.Meta != nil {
		for key := range e.original.Meta {
			e.markAsDeleted(metadataKeyPrefix + key)
		}
	}
	if e.original.Fields != nil {
		for key := range e.original.Fields {
			e.markAsDeleted(key)
		}
	}
}

// DeepUpdate recursively copies the key-value pairs from `d` to various properties of the event.
// When the key equals `@timestamp` it's set as the `Timestamp` property of the event.
// When the key equals `@metadata` the update is routed into the `Meta` map instead of `Fields`
// The rest of the keys are set to the `Fields` map.
// If the key is present and the value is a map as well, the sub-map will be updated recursively
// via `DeepUpdate`.
// `DeepUpdateNoOverwrite` is a version of this function that does not
// overwrite existing values.
func (e *EventEditor) DeepUpdate(d mapstr.M) { e.deepUpdate(d, updateModeOverwrite) }

// DeepUpdateNoOverwrite recursively copies the key-value pairs from `d` to various properties of the event.
// The `@timestamp` update is ignored due to "no overwrite" behavior.
// When the key equals `@metadata` the update is routed into the `Meta` map instead of `Fields`.
// The rest of the keys are set to the `Fields` map.
// If the key is present and the value is a map as well, the sub-map will be updated recursively
// via `DeepUpdateNoOverwrite`.
// `DeepUpdate` is a version of this function that overwrites existing values.
func (e *EventEditor) DeepUpdateNoOverwrite(d mapstr.M) { e.deepUpdate(d, updateModeNoOverwrite) }

// Apply write all the changes to the original event making sure that none of the original
// nested maps are modified but replaced with modified copies.
func (e *EventEditor) Apply() {
	defer e.Reset()

	for deletedKey := range e.deletions {
		_ = e.original.Delete(deletedKey)
	}

	if e.pending == nil {
		return
	}

	e.original.Timestamp = e.pending.Timestamp

	// it's enough to overwrite the root-level because
	// of the checkout mechanism used earlier
	if len(e.pending.Meta) > 0 {
		if e.original.Meta == nil {
			e.original.Meta = mapstr.M{}
		}
		for key := range e.pending.Meta {
			e.original.Meta[key] = e.pending.Meta[key]
		}
	}
	if len(e.pending.Fields) > 0 {
		if e.original.Fields == nil {
			e.original.Fields = mapstr.M{}
		}
		for key := range e.pending.Fields {
			e.original.Fields[key] = e.pending.Fields[key]
		}
	}
}

// Reset cleans all the pending changes and starts collecting them again.
// This function does not allocate new memory.
func (e *EventEditor) Reset() {
	if e.pending == nil {
		return
	}
	e.pending.Timestamp = e.original.Timestamp
	for k := range e.deletions {
		delete(e.deletions, k)
	}
	for k := range e.pending.Meta {
		delete(e.pending.Meta, k)
	}
	for k := range e.pending.Fields {
		delete(e.pending.Fields, k)
	}
}

// AddTags appends a tag to the tags field of the event. If the tags field does not
// exist then it will be created. If the tags field exists and is not a []string
// then an error will be returned. It does not deduplicate the list of tags.
func (e *EventEditor) AddTags(tags ...string) error {
	return e.AddTagsWithKey(mapstr.TagsKey, tags...)
}

// AddTagsWithKey appends a tag to the given field of the event. If the tags field does not
// exist then it will be created. If the tags field exists and is not a []string
// then an error will be returned. It does not deduplicate the list of tags.
func (e *EventEditor) AddTagsWithKey(key string, tags ...string) error {
	if len(tags) == 0 {
		return nil
	}
	e.checkout(key, checkoutModeIncludeValues)
	e.allocatePending()
	if e.pending.Fields == nil {
		e.pending.Fields = mapstr.M{}
	}
	return mapstr.AddTagsWithKey(e.pending.Fields, key, tags)
}

// AddError appends an error to the event. If the error field does not
// exist then it will be created.
// If the error field exists and another error was already set, it will be converted to a list
// of errors and new errors will be appended there.
// If there is only one error to add, it will be added directly without a list.
func (e *EventEditor) AddError(ee ...EventError) {
	if len(ee) == 0 {
		return
	}
	e.checkout(ErrorFieldKey, checkoutModeIncludeValues)
	e.allocatePending()
	if e.pending.Fields == nil {
		e.pending.Fields = mapstr.M{}
	}

	var list []mapstr.M
	val, err := e.pending.Fields.GetValue(ErrorFieldKey)
	// if the value does not exist yet, we initialize the list of errors
	// with the size of the current arguments.
	if errors.Is(err, mapstr.ErrKeyNotFound) {
		list = make([]mapstr.M, 0, len(ee))
	} else if err != nil {
		return
	}
	// if the value already exists, depending on its type
	// we need to reformat it or just append to an existing list.
	switch typed := val.(type) {
	// there was a single error map, should be replaced with a list,
	// the existing error will become an item on this list
	case mapstr.M:
		list = make([]mapstr.M, 0, len(ee)+1)
		list = append(list, typed)
	// same as the previous case but typed differently
	case map[string]interface{}:
		list = make([]mapstr.M, 0, len(ee)+1)
		list = append(list, mapstr.M(typed))
	// some code can assign an error as a string
	// it's not really expected but we should not lose this value
	// so, we convert it into a proper error format.
	case string:
		list = make([]mapstr.M, 0, len(ee)+1)
		list = append(list, EventError{Message: typed}.toMap())
	}

	// after the list is prepared and contains already existing errors
	// we can start copying the arguments, converting them to maps
	for _, err := range ee {
		// an error without a message is not a valid error
		if err.Message == "" {
			continue
		}
		m := err.toMap()
		list = append(list, m)
	}

	// if after all manipulations we still have a single error
	// we convert it back into a map instead of adding a list
	if len(list) == 1 {
		e.pending.Fields[ErrorFieldKey] = list[0]
	} else {
		e.pending.Fields[ErrorFieldKey] = list
	}
}

// String returns the string representation of the current editor's state.
// This function is slow and should be used for debugging purposes only.
func (e *EventEditor) String() string {
	deleteList := make([]string, 0, len(e.deletions))
	for key := range e.deletions {
		deleteList = append(deleteList, key)
	}
	m := mapstr.M{
		"original":  e.original.String(),
		"pending":   e.pending.String(),
		"deletions": deleteList,
	}
	return m.String()
}

// deepUpdate checks out all the necessary root-level keys before running the deep update.
func (e *EventEditor) deepUpdate(d mapstr.M, mode updateMode) {
	if len(d) == 0 {
		return
	}
	cm := checkoutModeOnlyMaps
	if mode == updateModeNoOverwrite {
		cm = checkoutModeIncludeValues
	}

	// checkout necessary keys from the original event
	for key := range d {
		if key == TimestampFieldKey {
			continue
		}

		// we never checkout the whole metadata, only root-level keys
		// which are about to get updated
		if key == MetadataFieldKey {
			metaUpdate := d[MetadataFieldKey]
			switch m := metaUpdate.(type) {
			case mapstr.M:
				for innerKey := range m {
					e.checkout(metadataKeyPrefix+innerKey, cm)
				}
			case map[string]interface{}:
				for innerKey := range m {
					e.checkout(metadataKeyPrefix+innerKey, cm)
				}
			}
			continue
		}

		e.checkout(key, cm)
	}

	e.allocatePending()
	e.pending.deepUpdate(d, mode)
}

// markAsDeleted marks a key for deletion when `Apply` is called
// if it's a root-level key in the original event.
func (e *EventEditor) markAsDeleted(key string) {
	if key == TimestampFieldKey {
		return
	}
	rootKey := e.rootKey(key)
	if e.deletions == nil {
		e.deletions = make(map[string]struct{})
	}
	e.deletions[rootKey] = struct{}{}
}

// checkout clones a nested map of the original event to the event with pending changes.
//
// If the key leads to a value nested in a map, we checkout the root-level nested map which means
// the whole sub-tree is recursively cloned, so it can be safely modified later.
func (e *EventEditor) checkout(key string, mode checkoutMode) {
	if key == TimestampFieldKey {
		return
	}
	// we're always looking only at the root-level
	rootKey := e.rootKey(key)
	// if the key is not in the original map, the value is empty
	if rootKey == "" {
		return
	}

	e.allocatePending()

	var dstMap, srcMap mapstr.M
	if strings.HasPrefix(rootKey, metadataKeyPrefix) {
		if e.original.Meta == nil {
			return
		}
		if e.pending.Meta == nil {
			e.pending.Meta = mapstr.M{}
		}
		dstMap = e.pending.Meta
		srcMap = e.original.Meta
		rootKey = rootKey[metadataKeyOffset:]
	} else {
		if e.original.Fields == nil {
			return
		}
		if e.pending.Fields == nil {
			e.pending.Fields = mapstr.M{}
		}
		dstMap = e.pending.Fields
		srcMap = e.original.Fields
	}

	// it might be already checked out
	_, checkedOut := dstMap[rootKey]
	if checkedOut {
		return
	}

	// if there is nothing to checkout - return
	value, exists := srcMap[rootKey]
	if !exists {
		return
	}

	// we check out only nested maps, and leave root-level value types in the original map
	// unless the special `includeValues` mode engaged (used for DeepUpdateNoOverwrite).
	switch typedVal := value.(type) {
	case mapstr.M:
		dstMap[rootKey] = typedVal.Clone()
	case map[string]interface{}:
		dstMap[rootKey] = mapstr.M(typedVal).Clone()
		// TODO slices?
	default:
		if mode == checkoutModeIncludeValues {
			dstMap[rootKey] = typedVal
		}
	}
}

// rootKey reduces the key of the original event to its root-level.
// Keys may have `.` character in them on every level and `.` can also mean a nested depth.
//
// Accounts for the `@metadata` subkeys, since it's stored in a separate map,
// root-level keys will be in the `@metadata.*` namespace.
//
// Returns empty string if the key does not exist in the original event.
func (e *EventEditor) rootKey(key string) string {
	if key == TimestampFieldKey {
		return key
	}
	var (
		prefix string
		m      mapstr.M
	)

	if strings.HasPrefix(key, metadataKeyPrefix) {
		prefix = metadataKeyPrefix
		key = key[metadataKeyOffset:]
		m = e.original.Meta
	} else {
		m = e.original.Fields
	}

	// there is no root-level key with this name
	if m == nil {
		return ""
	}

	// this key may be simple and does not contain dots
	dotIdx := strings.IndexRune(key, '.')
	if dotIdx == -1 {
		return prefix + key
	}

	// fast lane for the whole key
	_, exists := m[key]
	if exists {
		return prefix + key
	}

	// otherwise we start with the first segment
	// and keep adding the next segment to the key until the resulting value
	// exists on the root level.
	var rootKey string
	for {
		rootKey = key[:dotIdx]
		_, exists = m[rootKey]
		if exists {
			return prefix + rootKey
		}
		dotIdx = strings.IndexRune(key[dotIdx+1:], '.')
		// we checked above for the full key and it didn't exist,
		// no need to check again, we return since the key is not found
		if dotIdx == -1 {
			return ""
		}
		dotIdx += len(rootKey) + 1
	}
}

// checkDeleted returns `true` if the key was marked for deletion.
// The key can be expressed in dot-notation (e.g. x.y) and if the root-level prefix on
// the key path is deleted the function returns `true`.
func (e *EventEditor) checkDeleted(key string) bool {
	rootKey := e.rootKey(key)
	_, deleted := e.deletions[rootKey]
	return deleted
}

// allocatePending makes sure that the `pending` event is allocated for collecting new changes.
func (e *EventEditor) allocatePending() {
	if e.pending != nil {
		return
	}
	e.pending = &Event{
		Timestamp: e.original.Timestamp,
	}
}
