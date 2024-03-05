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
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type updateMode bool

var (
	updateModeOverwrite   updateMode = true
	updateModeNoOverwrite updateMode = false
)

// FlagField fields used to keep information or errors when events are parsed.
const FlagField = "log.flags"

const (
	TimestampFieldKey = "@timestamp"
	MetadataFieldKey  = "@metadata"
	ErrorFieldKey     = "error"
	metadataKeyPrefix = MetadataFieldKey + "."
	metadataKeyOffset = len(metadataKeyPrefix)
)

// Event is the common event format shared by all beats.
// Every event must have a timestamp and provide encodable Fields in `Fields`.
// The `Meta`-fields can be used to pass additional meta-data to the outputs.
// Output can optionally publish a subset of Meta, or ignore Meta.
type Event struct {
	Timestamp  time.Time
	Meta       mapstr.M
	Fields     mapstr.M
	Private    interface{} // for beats private use
	TimeSeries bool        // true if the event contains timeseries data
}

type PreEncoder interface {
	EncodeEvent(*Event) interface{}
}

type PreEncoderFactory func() PreEncoder

var (
	ErrValueNotTimestamp = errors.New("value is not a timestamp")
	ErrValueNotMapStr    = errors.New("value is not `mapstr.M` or `map[string]interface{}` type")
	ErrAlterMetadataKey  = fmt.Errorf("deleting/replacing %q key is not supported", MetadataFieldKey)
	ErrMetadataAccess    = fmt.Errorf("accessing %q key directly is not supported, try nested keys", MetadataFieldKey)
	ErrDeleteTimestamp   = fmt.Errorf("deleting %q key is not supported", TimestampFieldKey)
)

// SetID overwrites the "id" field in the events metadata.
// If Meta is nil, a new Meta dictionary is created.
func (e *Event) SetID(id string) {
	_, _ = e.PutValue(metadataKeyPrefix+"_id", id)
}

// GetValue gets a value from the event. If the key does not exist then an error
// is returned.
//
// Use `@timestamp` key for getting the event timestamp.
// Use `@metadata.*` keys for getting the event metadata fields.
// If `@metadata` key is used then `ErrMetadataAccess` is returned.
func (e *Event) GetValue(key string) (interface{}, error) {
	if key == TimestampFieldKey {
		return e.Timestamp, nil
	}
	if key == MetadataFieldKey {
		return nil, ErrMetadataAccess
	}

	if subKey, ok := e.metadataSubKey(key); ok {
		if e.Meta == nil {
			return nil, mapstr.ErrKeyNotFound
		}
		return e.Meta.GetValue(subKey)
	}

	if e.Fields == nil {
		return nil, mapstr.ErrKeyNotFound
	}

	return e.Fields.GetValue(key)
}

// Clone creates an exact copy of the event
func (e *Event) Clone() *Event {
	return &Event{
		Timestamp:  e.Timestamp,
		Meta:       e.Meta.Clone(),
		Fields:     e.Fields.Clone(),
		Private:    e.Private,
		TimeSeries: e.TimeSeries,
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
func (e *Event) DeepUpdate(d mapstr.M) {
	e.deepUpdate(d, updateModeOverwrite)
}

// DeepUpdateNoOverwrite recursively copies the key-value pairs from `d` to various properties of the event.
// The `@timestamp` update is ignored due to "no overwrite" behavior.
// When the key equals `@metadata` the update is routed into the `Meta` map instead of `Fields`.
// The rest of the keys are set to the `Fields` map.
// If the key is present and the value is a map as well, the sub-map will be updated recursively
// via `DeepUpdateNoOverwrite`.
// `DeepUpdate` is a version of this function that overwrites existing values.
func (e *Event) DeepUpdateNoOverwrite(d mapstr.M) {
	e.deepUpdate(d, updateModeNoOverwrite)
}

func (e *Event) deepUpdate(d mapstr.M, mode updateMode) {
	if len(d) == 0 {
		return
	}

	// It's supported to update the timestamp using this function.
	// However, we must handle it separately since it's a separate field of the event.
	timestampValue, timestampExists := d[TimestampFieldKey]
	if timestampExists {
		if mode == updateModeOverwrite {
			_, _ = e.setTimestamp(timestampValue)
		}

		// Temporary delete it from the update map,
		// so we can do `e.Fields.DeepUpdate(d)` or
		// `e.Fields.DeepUpdateNoOverwrite(d)` later
		delete(d, TimestampFieldKey)
		defer func() {
			d[TimestampFieldKey] = timestampValue
		}()
	}

	// It's supported to update the metadata using this function.
	// However, we must handle it separately since it's a separate field of the event.
	metaValue, metaExists := d[MetadataFieldKey]
	if metaExists {
		var metaUpdate mapstr.M

		switch meta := metaValue.(type) {
		case mapstr.M:
			metaUpdate = meta
		case map[string]interface{}:
			metaUpdate = mapstr.M(meta)
		}

		if metaUpdate != nil {
			if e.Meta == nil {
				e.Meta = mapstr.M{}
			}
			switch mode {
			case updateModeOverwrite:
				e.Meta.DeepUpdate(metaUpdate)
			case updateModeNoOverwrite:
				e.Meta.DeepUpdateNoOverwrite(metaUpdate)
			}
		}

		// Temporary delete it from the update map,
		// so we can do `e.Fields.DeepUpdate(d)` or
		// `e.Fields.DeepUpdateNoOverwrite(d)` later
		delete(d, MetadataFieldKey)
		defer func() {
			d[MetadataFieldKey] = metaValue
		}()
	}

	if len(d) == 0 {
		return
	}

	if e.Fields == nil {
		e.Fields = mapstr.M{}
	}

	switch mode {
	case updateModeOverwrite:
		e.Fields.DeepUpdate(d)
	case updateModeNoOverwrite:
		e.Fields.DeepUpdateNoOverwrite(d)
	}
}

func (e *Event) setTimestamp(v interface{}) (interface{}, error) {
	// to satisfy the PutValue interface, this function
	// must return the overwritten value
	prevValue := e.Timestamp

	switch ts := v.(type) {
	case time.Time:
		e.Timestamp = ts
		return prevValue, nil
	case common.Time:
		e.Timestamp = time.Time(ts)
		return prevValue, nil
	default:
		return nil, ErrValueNotTimestamp
	}
}

// Put associates the specified value with the specified key. If the event
// previously contained a mapping for the key, the old value is replaced and
// returned. The key can be expressed in dot-notation (e.g. x.y) to put a value
// into a nested map.
//
// If you need insert keys containing dots then you must use bracket notation
// to insert values (e.g. m[key] = value).
//
// Use `@timestamp` key for setting the event timestamp.
// Use `@metadata.*` keys for setting the event metadata fields.
// If `@metadata` key is used then `ErrAlterMetadataKey` is returned.
func (e *Event) PutValue(key string, v interface{}) (interface{}, error) {
	if key == TimestampFieldKey {
		return e.setTimestamp(v)
	}
	if key == MetadataFieldKey {
		return nil, ErrAlterMetadataKey
	}

	if subKey, ok := e.metadataSubKey(key); ok {
		if e.Meta == nil {
			e.Meta = mapstr.M{}
		}

		return e.Meta.Put(subKey, v)
	}

	if e.Fields == nil {
		e.Fields = mapstr.M{}
	}

	return e.Fields.Put(key, v)
}

// Delete deletes the given key from the event.
//
// Use `@metadata.*` keys for deleting the event metadata fields.
// If `@metadata` key is used then `ErrAlterMetadataKey` is returned.
// If `@timestamp` key is used then `ErrDeleteTimestamp` is returned.
func (e *Event) Delete(key string) error {
	if key == TimestampFieldKey {
		return ErrDeleteTimestamp
	}
	if key == MetadataFieldKey {
		return ErrAlterMetadataKey
	}
	if subKey, ok := e.metadataSubKey(key); ok {
		if e.Meta == nil {
			return mapstr.ErrKeyNotFound
		}
		return e.Meta.Delete(subKey)
	}

	if e.Fields == nil {
		return mapstr.ErrKeyNotFound
	}
	return e.Fields.Delete(key)
}

func (e *Event) metadataSubKey(key string) (string, bool) {
	if !strings.HasPrefix(key, metadataKeyPrefix) {
		return "", false
	}

	subKey := key[metadataKeyOffset:]
	if subKey == "" {
		return "", false
	}
	return subKey, true
}

// SetErrorWithOption sets the event error field with the message when the addErrKey is set to true.
// If you want to include the data and field you can pass them as parameters and will be appended into the
// error as fields with the corresponding name.
func (e *Event) SetErrorWithOption(message string, addErrKey bool, data string, field string) {
	if !addErrKey {
		return
	}

	errorField := mapstr.M{"message": message, "type": "json"}
	if data != "" {
		errorField["data"] = data
	}
	if field != "" {
		errorField["field"] = field
	}
	e.Fields[ErrorFieldKey] = errorField
}

// String returns a string representation of the event.
func (e *Event) String() string {
	m := mapstr.M{
		TimestampFieldKey: e.Timestamp,
		MetadataFieldKey:  mapstr.M{},
	}
	if e.Meta != nil {
		m[MetadataFieldKey] = e.Meta
	}
	m.DeepUpdate(e.Fields)
	return m.String()
}

// HasKey returns true if the key exist. If an error occurs then false is
// returned with a non-nil error.
func (e *Event) HasKey(key string) (bool, error) {
	if key == TimestampFieldKey || key == MetadataFieldKey {
		return true, nil
	}

	if subKey, ok := e.metadataSubKey(key); ok {
		if e.Meta == nil {
			return false, nil
		}
		return e.Meta.HasKey(subKey)
	}

	if e.Fields == nil {
		return false, nil
	}

	return e.Fields.HasKey(key)
}
