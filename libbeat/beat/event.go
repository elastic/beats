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
	}

	// TODO: add support to write into '@metadata'?
	return e.Fields.Put(key, v)
}

func (e *Event) Delete(key string) error {
	return e.Fields.Delete(key)
}
