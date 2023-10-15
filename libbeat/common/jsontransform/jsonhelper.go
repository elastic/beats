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

package jsontransform

import (
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
)

const (
	iso8601 = "2006-01-02T15:04:05.000Z0700"
)

var (
	// ErrInvalidTimestamp is returned when parsing of a @timestamp field fails.
	// Supported formats: ISO8601, RFC3339
	ErrInvalidTimestamp = errors.New("failed to parse @timestamp, unknown format")
)

// WriteJSONKeys writes the json keys to the given event based on the overwriteKeys option and the addErrKey
func WriteJSONKeys(event *beat.EventEditor, keys map[string]interface{}, expandKeys, overwriteKeys, addErrKey bool) {
	setError := func(string, string) {}
	if addErrKey {
		setError = func(msg, field string) {
			event.AddError(beat.EventError{Message: msg, Field: field})
		}
	}
	if expandKeys {
		if err := expandFields(keys); err != nil {
			setError(err.Error(), "")
			return
		}
	}
	if !overwriteKeys {
		// Then, perform deep update without overwriting
		event.DeepUpdateNoOverwrite(keys)
		return
	}

	for k, v := range keys {
		switch k {
		case beat.TimestampFieldKey:
			vstr, ok := v.(string)
			if !ok {
				setError("not overwritten (not a string)", beat.TimestampFieldKey)
				removeKeys(keys, k)
				continue
			}

			// @timestamp must be of format RFC3339 or ISO8601
			ts, err := parseTimestamp(vstr)
			if err != nil {
				setError(fmt.Sprintf("timestamp parse error on %s", vstr), beat.TimestampFieldKey)
				removeKeys(keys, k)
				continue
			}
			keys[k] = ts
		case beat.MetadataFieldKey:
			switch v.(type) {
			case map[string]string, map[string]interface{}:
			default:
				setError("can't replace with a value", beat.MetadataFieldKey)
				removeKeys(keys, k)
			}

		case beat.TypeFieldKey:
			vstr, ok := v.(string)
			if !ok {
				setError("not overwritten (not a string)", beat.TypeFieldKey)
				removeKeys(keys, k)
				continue
			}
			if len(vstr) == 0 || vstr[0] == '_' {
				setError(fmt.Sprintf("not overwritten (invalid value [%s])", vstr), beat.TypeFieldKey)
				removeKeys(keys, k)
				continue
			}
			keys[k] = vstr
		}
	}

	event.DeepUpdate(keys)
}

func removeKeys(keys map[string]interface{}, names ...string) {
	for _, name := range names {
		delete(keys, name)
	}
}

func parseTimestamp(timestamp string) (time.Time, error) {
	validFormats := []string{
		time.RFC3339,
		iso8601,
	}

	for _, f := range validFormats {
		ts, parseErr := time.Parse(f, timestamp)
		if parseErr != nil {
			continue
		}

		return ts, nil
	}

	return time.Time{}, ErrInvalidTimestamp
}
