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

package conditions

import (
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func BenchmarkSimpleCondition(b *testing.B) {
	config := Config{
		HasFields: []string{"afield"},
	}

	cond, err := NewCondition(&config)
	if err != nil {
		panic(err)
	}

	event := &beat.Event{
		Timestamp: time.Now(),
		Fields: mapstr.M{
			"@timestamp": "2015-06-11T09:51:23.642Z",
			"afield":     "avalue",
		},
	}

	for i := 0; i < b.N; i++ {
		cond.Check(event)
	}
}

func BenchmarkCombinedCondition(b *testing.B) {
	config := Config{
		OR: []Config{
			{
				Range: &Fields{fields: map[string]interface{}{
					"http.code.gte": 100,
					"http.code.lt":  300,
				}},
			},
			{
				AND: []Config{
					{
						Equals: &Fields{fields: map[string]interface{}{
							"status": 200,
						}},
					},
					{
						Equals: &Fields{fields: map[string]interface{}{
							"type": "http",
						}},
					},
				},
			},
		},
	}

	cond, err := NewCondition(&config)
	if err != nil {
		panic(err)
	}

	event := &beat.Event{
		Timestamp: time.Now(),
		Fields: mapstr.M{
			"@timestamp":    "2015-06-11T09:51:23.642Z",
			"bytes_in":      126,
			"bytes_out":     28033,
			"client_ip":     "127.0.0.1",
			"client_port":   42840,
			"client_proc":   "",
			"client_server": "mar.local",
			"http": mapstr.M{
				"code":           200,
				"content_length": 76985,
				"phrase":         "OK",
			},
			"ip":           "127.0.0.1",
			"method":       "GET",
			"params":       "",
			"path":         "/jszip.min.js",
			"port":         8000,
			"proc":         "",
			"query":        "GET /jszip.min.js",
			"responsetime": 30,
			"server":       "mar.local",
			"status":       "OK",
			"type":         "http",
		},
	}

	for i := 0; i < b.N; i++ {
		cond.Check(event)
	}
}
