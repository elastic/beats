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

package cbortransform

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

// WriteCBORKeys writes the cbor keys to the given event based on the overwriteKeys option
func WriteCBORKeys(event *beat.Event, keys map[string]interface{}, overwriteKeys bool) {
	if !overwriteKeys {
		for k, v := range keys {
			if _, exists := event.Fields[k]; !exists && k != "@timestamp" && k != "@metadata" {
				event.Fields[k] = v
			}
		}
		return
	}

	for k, v := range keys {
		switch k {
		case "@timestamp":
			vstr, ok := v.(string)
			if !ok {
				logp.Err("CBOR: Won't overwrite @timestamp because value is not string")
				event.Fields["error"] = createCBORError("@timestamp not overwritten (not string)")
				continue
			}

			// @timestamp must be of format RFC3339
			ts, err := time.Parse(time.RFC3339, vstr)
			if err != nil {
				logp.Err("CBOR: Won't overwrite @timestamp because of parsing error: %v", err)
				event.Fields["error"] = createCBORError(fmt.Sprintf("@timestamp not overwritten (parse error on %s)", vstr))
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
				event.Meta.Update(common.MapStr(m))

			default:
				event.Fields["error"] = createCBORError("failed to update @metadata")
			}

		case "type":
			vstr, ok := v.(string)
			if !ok {
				logp.Err("CBOR: Won't overwrite type because value is not string")
				event.Fields["error"] = createCBORError("type not overwritten (not string)")
				continue
			}
			if len(vstr) == 0 || vstr[0] == '_' {
				logp.Err("CBOR: Won't overwrite type because value is empty or starts with an underscore")
				event.Fields["error"] = createCBORError(fmt.Sprintf("type not overwritten (invalid value [%s])", vstr))
				continue
			}
			event.Fields[k] = vstr

		default:
			event.Fields[k] = v
		}
	}
}

func createCBORError(message string) common.MapStr {
	return common.MapStr{"message": message, "type": "cbor"}
}
