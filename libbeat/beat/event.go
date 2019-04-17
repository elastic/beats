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

	"github.com/elastic/beats/libbeat/common"
)

// FlagField fields used to keep information or errors when events are parsed.
const FlagField = "log.flags"

// Event is the common event format shared by all beats.
// Every event must have a timestamp and provide encodable Fields in `Fields`.
// The `Meta`-fields can be used to pass additional meta-data to the outputs.
// Output can optionally publish a subset of Meta, or ignore Meta.
type Event struct {
	Timestamp time.Time
	Meta      common.MapStr
	Fields    common.MapStr
	Private   interface{} // for beats private use
}

var (
	errNoTimestamp = errors.New("value is no timestamp")
	errNoMapStr    = errors.New("value is no map[string]interface{} type")
)

// SetID overwrites the "id" field in the events metadata.
// If Meta is nil, a new Meta dictionary is created.
func (e *Event) SetID(id string) {
	if e.Meta == nil {
		e.Meta = common.MapStr{}
	}
	e.Meta["id"] = id
}

func (e *Event) GetValue(key string) (interface{}, error) {
	if key == "@timestamp" {
		return e.Timestamp, nil
	} else if subKey, ok := metadataKey(key); ok {
		if subKey == "" || e.Meta == nil {
			return e.Meta, nil
		}
		return e.Meta.GetValue(subKey)
	}
	return e.Fields.GetValue(key)
}

func (e *Event) PutValue(key string, v interface{}) (interface{}, error) {
	if key == "@timestamp" {
		switch ts := v.(type) {
		case time.Time:
			e.Timestamp = ts
		case common.Time:
			e.Timestamp = time.Time(ts)
		default:
			return nil, errNoTimestamp
		}
	} else if subKey, ok := metadataKey(key); ok {
		if subKey == "" {
			switch meta := v.(type) {
			case common.MapStr:
				e.Meta = meta
			case map[string]interface{}:
				e.Meta = meta
			default:
				return nil, errNoMapStr
			}
		} else if e.Meta == nil {
			e.Meta = common.MapStr{}
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
	if !strings.HasPrefix(key, "@metadata") {
		return "", false
	}

	subKey := key[len("@metadata"):]
	if subKey == "" {
		return "", true
	}
	if subKey[0] == '.' {
		return subKey[1:], true
	}
	return "", false
}
