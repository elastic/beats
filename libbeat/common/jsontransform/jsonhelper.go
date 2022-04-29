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
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
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
func WriteJSONKeys(event *beat.Event, keys map[string]interface{}, expandKeys, overwriteKeys, addErrKey bool) {
	logger := logp.NewLogger("jsonhelper")
	if expandKeys {
		if err := expandFields(keys); err != nil {
			logger.Errorf("JSON: failed to expand fields: %s", err)
			event.SetErrorWithOption(createJSONError(err.Error()), addErrKey)
			return
		}
	}
	if !overwriteKeys {
		// @timestamp and @metadata fields are root-level fields. We remove them so they
		// don't become part of event.Fields.
		removeKeys(keys, "@timestamp", "@metadata")

		// Then, perform deep update without overwriting
		event.Fields.DeepUpdateNoOverwrite(keys)
		return
	}

	for k, v := range keys {
		switch k {
		case "@timestamp":
			vstr, ok := v.(string)
			if !ok {
				logger.Error("JSON: Won't overwrite @timestamp because value is not string")
				event.SetErrorWithOption(createJSONError("@timestamp not overwritten (not string)"), addErrKey)
				continue
			}

			// @timestamp must be of format RFC3339 or ISO8601
			ts, err := parseTimestamp(vstr)
			if err != nil {
				logger.Errorf("JSON: Won't overwrite @timestamp because of parsing error: %v", err)
				event.SetErrorWithOption(createJSONError(fmt.Sprintf("@timestamp not overwritten (parse error on %s)", vstr)), addErrKey)
				continue
			}
			event.Timestamp = ts

		case "@metadata":
			switch m := v.(type) {
			case map[string]string:
				if event.Meta == nil && len(m) > 0 {
					event.Meta = mapstr.M{}
				}
				for meta, value := range m {
					event.Meta[meta] = value
				}

			case map[string]interface{}:
				if event.Meta == nil {
					event.Meta = mapstr.M{}
				}
				event.Meta.DeepUpdate(mapstr.M(m))

			default:
				event.SetErrorWithOption(createJSONError("failed to update @metadata"), addErrKey)
			}

		case "type":
			vstr, ok := v.(string)
			if !ok {
				logger.Error("JSON: Won't overwrite type because value is not string")
				event.SetErrorWithOption(createJSONError("type not overwritten (not string)"), addErrKey)
				continue
			}
			if len(vstr) == 0 || vstr[0] == '_' {
				logger.Error("JSON: Won't overwrite type because value is empty or starts with an underscore")
				event.SetErrorWithOption(createJSONError(fmt.Sprintf("type not overwritten (invalid value [%s])", vstr)), addErrKey)
				continue
			}
			event.Fields[k] = vstr
		}
	}

	// We have accounted for @timestamp, @metadata, type above. So let's remove these keys and
	// deep update the event with the rest of the keys.
	removeKeys(keys, "@timestamp", "@metadata", "type")
	event.Fields.DeepUpdate(keys)
}

func createJSONError(message string) mapstr.M {
	return mapstr.M{"message": message, "type": "json"}
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
