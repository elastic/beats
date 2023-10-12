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
	"strings"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

type checkoutMode bool

var (
	checkoutModeOnlyMaps      checkoutMode = false
	checkoutModeIncludeValues checkoutMode = true
)

// TODO rename to `Event` and use it instead of the actual event struct everywhere
type EventAccessor interface {
	// GetValue gets a value from the map. The key can be expressed in dot-notation (e.g. x.y).
	// If the key does not exist then `mapstr.ErrKeyNotFound` error is returned.
	GetValue(key string) (interface{}, error)

	// PutValue associates the specified value with the specified key. If the event
	// previously contained a mapping for the key, the old value is replaced and
	// returned. The key can be expressed in dot-notation (e.g. x.y) to put a value
	// into a nested map.
	//
	// If you need insert keys containing dots then you must use bracket notation
	// to insert values (e.g. m[key] = value).
	PutValue(key string, v interface{}) (interface{}, error)

	// Delete value with the given key.
	// The key can be expressed in dot-notation (e.g. x.y)
	Delete(key string) error

	// DeepUpdate recursively copies the key-value pairs from `d` to various properties of the event.
	// When the key equals `@timestamp` it's set as the `Timestamp` property of the event.
	// When the key equals `@metadata` the update is routed into the `Meta` map instead of `Fields`
	// The rest of the keys are set to the `Fields` map.
	// If the key is present and the value is a map as well, the sub-map will be updated recursively
	// via `DeepUpdate`.
	// `DeepUpdateNoOverwrite` is a version of this function that does not
	// overwrite existing values.p
	DeepUpdate(d mapstr.M)

	// DeepUpdateNoOverwrite recursively copies the key-value pairs from `d` to various properties of the event.
	// The `@timestamp` update is ignored due to "no overwrite" behavior.
	// When the key equals `@metadata` the update is routed into the `Meta` map instead of `Fields`.
	// The rest of the keys are set to the `Fields` map.
	// If the key is present and the value is a map as well, the sub-map will be updated recursively
	// via `DeepUpdateNoOverwrite`.
	// `DeepUpdate` is a version of this function that overwrites existing values.
	DeepUpdateNoOverwrite(d mapstr.M)
}

// EventEditor is a wrapper that allows to make changes to the wrapped event
// preserving states of the original nested maps by cloning them on demand.
//
// The first time a nested map gets updated it's cloned and, from that moment on, only the copy is modified.
// Once all the changes are collected, users should call `Apply` to copy pending changes to the original event.
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

func NewEventEditor(e *Event) *EventEditor {
	if e == nil {
		e = &Event{}
	}
	return &EventEditor{
		original: e,
	}
}

// GetValue implements the `EventAccessor` interface.
func (e *EventEditor) GetValue(key string) (interface{}, error) {
	if key == metadataFieldKey {
		return nil, ErrMetadataAccess
	}

	// handle the deletion marks
	rootKey := e.rootKey(key)
	if rootKey == key && e.checkDeleted(key) {
		return nil, mapstr.ErrKeyNotFound
	}

	if e.pending != nil {
		// We try to retrieve from the `pending` event first,
		// since it should have the most recent data.
		//
		// To check if the nested map was checked-out before
		// we need to get the root-level first
		val, err := e.pending.GetValue(rootKey)
		if err == nil {
			if rootKey == key {
				// the value might be the end value we're looking for
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

// PutValue implements the `EventAccessor` interface.
func (e *EventEditor) PutValue(key string, v interface{}) (interface{}, error) {
	if key == metadataFieldKey {
		return nil, ErrAlterMetadataKey
	}
	// checkout only if the key leads to a nested value
	e.checkout(key, checkoutModeOnlyMaps)
	e.allocatePending()
	return e.pending.PutValue(key, v)
}

// Delete implements the `EventAccessor` interface.
func (e *EventEditor) Delete(key string) error {
	if key == timestampFieldKey {
		return ErrDeleteTimestamp
	}
	if key == metadataFieldKey {
		return ErrAlterMetadataKey
	}
	var deleted bool
	has, _ := e.original.HasKey(key)

	if has {
		if key == e.rootKey(key) {
			// if we're trying to delete a root-level key
			// it must be deleted from the `original` event when `Apply` is called
			// and from the `pending` event (below) if a value with the same key was put there.
			e.markAsDeleted(key)
			deleted = true
		} else {
			// if it's not a root-level key, we checkout this entire root-level map
			// and delete from the `pending` event instead.
			e.checkout(key, checkoutModeOnlyMaps)
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

// DeepUpdate implements the `EventAccessor` interface.
func (e *EventEditor) DeepUpdate(d mapstr.M) { e.deepUpdate(d, updateModeOverwrite) }

// DeepUpdateNoOverwrite implements the `EventAccessor` interface.
func (e *EventEditor) DeepUpdateNoOverwrite(d mapstr.M) { e.deepUpdate(d, updateModeNoOverwrite) }

// Apply write all the changes to the original event making sure that none of the original
// nested maps are modified but replaced with modified clones.
func (e *EventEditor) Apply() {
	if e.pending == nil {
		return
	}

	defer e.Reset()

	e.original.Timestamp = e.pending.Timestamp
	for deletedKey := range e.deletions {
		_ = e.original.Delete(deletedKey)
	}

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
		if key == timestampFieldKey {
			continue
		}

		// we never checkout the whole metadata, only root-level keys
		// which are about to get updated
		if key == metadataFieldKey {
			metaUpdate := d[metadataFieldKey]
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
	dotIdx := e.dotIdx(key)
	// nested keys are not marked since nested maps
	// are cloned into the `pending` event and altered there
	if dotIdx != -1 {
		return
	}
	if e.deletions == nil {
		e.deletions = make(map[string]struct{})
	}
	e.deletions[key] = struct{}{}
}

// checkout clones a nested map of the original event to the event with pending changes.
//
// If the key leads to a value nested in a map, we checkout the root-level nested map which means
// the whole sub-tree is recursively cloned, so it can be safely modified.
func (e *EventEditor) checkout(key string, mode checkoutMode) {
	// we're always looking only at the root-level
	rootKey := e.rootKey(key)

	if e.pending != nil {
		// it might be already checked out
		checkedOut, _ := e.pending.HasKey(rootKey)
		if checkedOut {
			return
		}
	}

	// if there is nothing to checkout - return
	value, err := e.original.GetValue(rootKey)
	if err != nil {
		return
	}

	e.allocatePending()

	// we check out only nested maps, and leave root-level value types in the original map
	// unless the special `includeValues` mode engaged (used for DeepUpdateNoOverwrite).
	switch typedVal := value.(type) {
	case mapstr.M:
		_, _ = e.pending.PutValue(rootKey, typedVal.Clone())
	case map[string]interface{}:
		_, _ = e.pending.PutValue(rootKey, mapstr.M(typedVal).Clone())
	default:
		if mode == checkoutModeIncludeValues {
			_, _ = e.pending.PutValue(rootKey, typedVal)
		}
	}
}

// dotIdx returns index of the first `.` character or `-1` if there is no `.` character.
// Accounts for the `@metadata` subkeys, since it's stored in a separate map,
// root-level keys will be in the `@metadata.*` namespace.
func (e *EventEditor) dotIdx(key string) int {
	// metadata keys are special, since they're stored in a separate map
	// we don't want to copy the whole map with all metadata, we want
	// to checkout only nested maps one by one for efficiency
	if strings.HasPrefix(key, metadataKeyPrefix) {
		// we start looking after the `@metadata` prefix
		dotIdx := strings.Index(key[metadataKeyOffset:], ".")
		// if there is no dot in the subkey, then the second segment
		// is considered to be a root-level key
		if dotIdx == -1 {
			return -1
		}
		// otherwise we need to offset the dot index by the prefix we removed
		return dotIdx + metadataKeyOffset
	}

	return strings.Index(key, ".")
}

// rootKey reduces the key to its root-level.
func (e *EventEditor) rootKey(key string) string {
	dotIdx := e.dotIdx(key)
	if dotIdx == -1 {
		return key
	} else {
		return key[:dotIdx]
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

// allocatePending makes sure that the `pending` event is allocated for collecting changes.
func (e *EventEditor) allocatePending() {
	if e.pending != nil {
		return
	}
	e.pending = &Event{
		Timestamp: e.original.Timestamp,
	}
}
