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
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// FlagField fields used to keep information or errors when events are parsed.
const FlagField = "log.flags"

const (
	timestampFieldKey = "@timestamp"
	metadataFieldKey  = "@metadata"
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

var (
	errNoTimestamp = errors.New("value is no timestamp")
	errNoMapStr    = errors.New("value is no map[string]interface{} type")
)

// SetID overwrites the "id" field in the events metadata.
// If Meta is nil, a new Meta dictionary is created.
func (e *Event) SetID(id string) {
	if e.Meta == nil {
		e.Meta = mapstr.M{}
	}
	e.Meta["_id"] = id
}

func (e *Event) GetValue(key string) (interface{}, error) {
	if key == timestampFieldKey {
		return e.Timestamp, nil
	} else if subKey, ok := metadataKey(key); ok {
		if subKey == "" || e.Meta == nil {
			return e.Meta, nil
		}
		return e.Meta.GetValue(subKey)
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
	e.deepUpdate(d, true)
}

// DeepUpdateNoOverwrite recursively copies the key-value pairs from `d` to various properties of the event.
// The `@timestamp` update is ignored due to "no overwrite" behavior.
// When the key equals `@metadata` the update is routed into the `Meta` map instead of `Fields`.
// The rest of the keys are set to the `Fields` map.
// If the key is present and the value is a map as well, the sub-map will be updated recursively
// via `DeepUpdateNoOverwrite`.
// `DeepUpdate` is a version of this function that overwrites existing values.
func (e *Event) DeepUpdateNoOverwrite(d mapstr.M) {
	e.deepUpdate(d, false)
}

func (e *Event) deepUpdate(d mapstr.M, overwrite bool) {
	if len(d) == 0 {
		return
	}

	// It's supported to update the timestamp using this function.
	// However, we must handle it separately since it's a separate field of the event.
	timestampValue, timestampExists := d[timestampFieldKey]
	if timestampExists {
		if overwrite {
			_ = e.setTimestamp(timestampValue)
		}

		// Temporary delete it from the update map,
		// so we can do `e.Fields.DeepUpdate(d)` or
		// `e.Fields.DeepUpdateNoOverwrite(d)` later
		delete(d, timestampFieldKey)
	}

	// It's supported to update the metadata using this function.
	// However, we must handle it separately since it's a separate field of the event.
	metaValue, metaExists := d[metadataFieldKey]
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
			if overwrite {
				e.Meta.DeepUpdate(metaUpdate)
			} else {
				e.Meta.DeepUpdateNoOverwrite(metaUpdate)
			}
		}

		// Temporary delete it from the update map,
		// so we can do `e.Fields.DeepUpdate(d)` or
		// `e.Fields.DeepUpdateNoOverwrite(d)` later
		delete(d, metadataFieldKey)
	}

	// At the end we revert all changes we made to the update map
	defer func() {
		if timestampExists {
			d[timestampFieldKey] = timestampValue
		}
		if metaExists {
			d[metadataFieldKey] = metaValue
		}
	}()

	if len(d) == 0 {
		return
	}

	if e.Fields == nil {
		e.Fields = mapstr.M{}
	}

	if overwrite {
		e.Fields.DeepUpdate(d)
	} else {
		e.Fields.DeepUpdateNoOverwrite(d)
	}
}

func (e *Event) setTimestamp(v interface{}) error {
	switch ts := v.(type) {
	case time.Time:
		e.Timestamp = ts
	case common.Time:
		e.Timestamp = time.Time(ts)
	default:
		return errNoTimestamp
	}

	return nil
}

func (e *Event) PutValue(key string, v interface{}) (interface{}, error) {
	if key == timestampFieldKey {
		err := e.setTimestamp(v)
		return nil, err
	} else if subKey, ok := metadataKey(key); ok {
		if subKey == "" {
			switch meta := v.(type) {
			case mapstr.M:
				e.Meta = meta
			case map[string]interface{}:
				e.Meta = meta
			default:
				return nil, errNoMapStr
			}
		} else if e.Meta == nil {
			e.Meta = mapstr.M{}
		}
		return e.Meta.Put(subKey, v)
	}

	return e.Fields.Put(key, v)
}

func (e *Event) Delete(key string) error {
	if subKey, ok := metadataKey(key); ok {
		if subKey == "" {
			e.Meta = nil
			return nil
		}
		if e.Meta == nil {
			return nil
		}
		return e.Meta.Delete(subKey)
	}
	return e.Fields.Delete(key)
}

func metadataKey(key string) (string, bool) {
	if !strings.HasPrefix(key, metadataFieldKey) {
		return "", false
	}

	subKey := key[len(metadataFieldKey):]
	if subKey == "" {
		return "", true
	}
	if subKey[0] == '.' {
		return subKey[1:], true
	}
	return "", false
}

// SetErrorWithOption sets the event error field with the message when the addErrKey is set to true.
// If you want to include the data and field you can pass them as parameters and will be appended into the
// error as fields with the corresponding name.
func (e *Event) SetErrorWithOption(message string, addErrKey bool, data string, field string) {
	if addErrKey {
		errorField := mapstr.M{"message": message, "type": "json"}
		if data != "" {
			errorField["data"] = data
		}
		if field != "" {
			errorField["field"] = field
		}
		e.Fields["error"] = errorField
	}
}
