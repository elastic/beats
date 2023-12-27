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
	"strings"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs/codec/json"
	"github.com/elastic/elastic-agent-libs/logp"
)

func makeLogstashEventEncoder(log *logp.Logger, info beat.Info, escapeHTML bool, index string) func(interface{}) ([]byte, error) {
	enc := json.New(info.Version, json.Config{
		Pretty:     false,
		EscapeHTML: escapeHTML,
	})
	index = strings.ToLower(index)
	return func(event interface{}) (d []byte, err error) {
		d, err = enc.Encode(index, event.(*beat.Event))
		if err != nil {
			log.Debugf("Failed to encode event: %v", event)
		}
		return
	}
}
