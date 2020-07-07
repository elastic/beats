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
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

// WriteJSONKeys writes the json keys to the given event based on the overwriteKeys option and the addErrKey
func WriteJSONKeys(event *beat.Event, keys map[string]interface{}, overwriteKeys bool, addErrKey bool) {
	logger := logp.NewLogger("jsonhelper")
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

			// @timestamp must be of format RFC3339
			ts, err := time.Parse(time.RFC3339, vstr)
			if err != nil {
				logger.Errorf("JSON: Won't overwrite @timestamp because of parsing error: %v", err)
				event.SetErrorWithOption(createJSONError(fmt.Sprintf("@timestamp not overwritten (parse error on %s)", vstr)), addErrKey)
				continue
			}
			event.Timestamp = ts

		case "@metadata":
			switch m := v.(type) {
			case map[string]string:
				for meta, value := range m {
					event.Meta[meta] = value
				}

			case map[string]interface{}:
				event.Meta.DeepUpdate(common.MapStr(m))

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

func createJSONError(message string) common.MapStr {
	return common.MapStr{"message": message, "type": "json"}
}

func removeKeys(keys map[string]interface{}, names ...string) {
	for _, name := range names {
		delete(keys, name)
	}
}
