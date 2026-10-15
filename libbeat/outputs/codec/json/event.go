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
	"maps"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Event describes the event structure for events
// (in-)directly send to logstash
type event struct {
	Timestamp time.Time `struct:"@timestamp"`
	Meta      meta      `struct:"@metadata"`
	Fields    mapstr.M  `struct:",inline"`
}

// Meta defines common event metadata to be stored in '@metadata'
type meta struct {
	Beat    string         `struct:"beat"`
	Type    string         `struct:"type"`
	Version string         `struct:"version"`
	Fields  map[string]any `struct:",inline"`
}

func makeEvent(index, version string, in *beat.Event) event {
	m := meta{
		Beat:    index,
		Version: version,
		Type:    "_doc",
		Fields:  in.Meta,
	}

	// '@metadata.beat' set on the event takes precedence over the output's index, so
	// events from several beats can share one output. It is removed from a
	// shallow copy of the inline map so it is not emitted twice.
	if b, ok := in.Meta["beat"].(string); ok && b != "" {
		m.Beat = b
		metaClone := maps.Clone(in.Meta)
		delete(metaClone, "beat")
		m.Fields = metaClone
	}

	return event{
		Timestamp: in.Timestamp,
		Meta:      m,
		Fields:    in.Fields,
	}
}
