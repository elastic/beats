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

package logstash

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/outputs/codec/json"
)

func makeLogstashEventEncoder(info beat.Info, escapeHTML bool, index string) func(interface{}) ([]byte, error) {
	enc := json.New(false, escapeHTML, info.Version)
	return func(event interface{}) (d []byte, err error) {
		d, err = enc.Encode(index, event.(*beat.Event))
		if err != nil {
			debugf("Failed to encode event: %v", event)
		}
		return
	}
}
