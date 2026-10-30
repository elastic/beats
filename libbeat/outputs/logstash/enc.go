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
	"errors"
	"maps"
	"strings"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs/codec/json"
	"github.com/elastic/elastic-agent-libs/logp"
)

func makeLogstashEventEncoder(log *logp.Logger, beatVersion string, escapeHTML bool, index string) func(any) ([]byte, error) {
	enc := json.New(beatVersion, json.Config{
		Pretty:     false,
		EscapeHTML: escapeHTML,
	})
	index = strings.ToLower(index)
	return func(event any) ([]byte, error) {
		beatEvent, ok := event.(*beat.Event)
		if !ok {
			return nil, errors.New("event is not of type *beat.Event")
		}

		eventIndex := index
		// '@metadata.beat' set on the event takes precedence over the output's index so
		// events from several beats can share one output. It is removed from a copy of
		// Meta so the encoder does not emit the key twice.
		if b, ok := beatEvent.Meta["beat"].(string); ok && b != "" {
			eventIndex = b
			e := *beatEvent
			e.Meta = maps.Clone(beatEvent.Meta)
			delete(e.Meta, "beat")
			beatEvent = &e
		}

		d, err := enc.Encode(eventIndex, beatEvent)
		if err != nil {
			log.Debugf("Failed to encode event: %v", event)
			return nil, err
		}
		return d, nil
	}
}
