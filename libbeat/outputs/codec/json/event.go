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

package json

import (
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

// Event describes the event structure for events
// (in-)directly send to logstash
type event struct {
	Timestamp time.Time     `struct:"@timestamp"`
	Meta      meta          `struct:"@metadata"`
	Fields    common.MapStr `struct:",inline"`
}

// Meta defines common event metadata to be stored in '@metadata'
type meta struct {
	Beat    string                 `struct:"beat"`
	Type    string                 `struct:"type"`
	Version string                 `struct:"version"`
	Fields  map[string]interface{} `struct:",inline"`
}

func makeEvent(index, version string, in *beat.Event) event {
	return event{
		Timestamp: in.Timestamp,
		Meta: meta{
			Beat:    index,
			Version: version,
			Type:    "_doc",
			Fields:  in.Meta,
		},
		Fields: in.Fields,
	}
}
