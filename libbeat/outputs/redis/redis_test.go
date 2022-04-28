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

package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/outputs"
	_ "github.com/elastic/beats/v7/libbeat/outputs/codec/json"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type checker func(*testing.T, outputs.Group)

func checks(cs ...checker) checker {
	return func(t *testing.T, g outputs.Group) {
		for _, c := range cs {
			c(t, g)
		}
	}
}

func clientsLen(required int) checker {
	return func(t *testing.T, group outputs.Group) {
		assert.Len(t, group.Clients, required)
	}
}

func clientPassword(index int, pass string) checker {
	return func(t *testing.T, group outputs.Group) {
		redisClient := group.Clients[index].(*backoffClient)
		assert.Equal(t, redisClient.client.password, pass)
	}
}

func TestMakeRedis(t *testing.T) {
	tests := map[string]struct {
		config map[string]interface{}
		valid  bool
		checks checker
	}{
		"no host": {
			config: map[string]interface{}{
				"hosts": []string{},
			},
		},
		"invald scheme": {
			config: map[string]interface{}{
				"hosts": []string{"redisss://localhost:6379"},
			},
		},
		"Single host": {
			config: map[string]interface{}{
				"hosts": []string{"localhost:6379"},
			},
			valid:  true,
			checks: checks(clientsLen(1), clientPassword(0, "")),
		},
		"Multiple hosts": {
			config: map[string]interface{}{
				"hosts": []string{"redis://localhost:6379", "rediss://localhost:6380"},
			},
			valid:  true,
			checks: clientsLen(2),
		},
		"Default password": {
			config: map[string]interface{}{
				"hosts":    []string{"redis://localhost:6379"},
				"password": "defaultPassword",
			},
			valid:  true,
			checks: checks(clientsLen(1), clientPassword(0, "defaultPassword")),
		},
		"Specific and default password": {
			config: map[string]interface{}{
				"hosts":    []string{"redis://localhost:6379", "rediss://:mypassword@localhost:6380"},
				"password": "defaultPassword",
			},
			valid: true,
			checks: checks(
				clientsLen(2),
				clientPassword(0, "defaultPassword"),
				clientPassword(1, "mypassword"),
			),
		},
	}
	beatInfo := beat.Info{Beat: "libbeat", Version: "1.2.3"}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg, err := common.NewConfigFrom(test.config)
			assert.NoError(t, err)
			groups, err := makeRedis(nil, beatInfo, outputs.NewNilObserver(), cfg)
			assert.Equal(t, err == nil, test.valid)
			if err != nil && test.valid {
				t.Log(err)
			}
			if test.checks != nil {
				test.checks(t, groups)
			}
		})
	}
}

func TestKeySelection(t *testing.T) {
	cases := map[string]struct {
		cfg   map[string]interface{}
		event beat.Event
		want  string
	}{
		"key configured": {
			cfg:  map[string]interface{}{"key": "test"},
			want: "test",
		},
		"key must keep case": {
			cfg:  map[string]interface{}{"key": "Test"},
			want: "Test",
		},
		"key setting": {
			cfg: map[string]interface{}{
				"keys": []map[string]interface{}{{"key": "test"}},
			},
			want: "test",
		},
		"keys setting must keep case": {
			cfg: map[string]interface{}{
				"keys": []map[string]interface{}{{"key": "Test"}},
			},
			want: "Test",
		},
		"use event field": {
			cfg: map[string]interface{}{"key": "test-%{[field]}"},
			event: beat.Event{
				Fields: mapstr.M{"field": "from-event"},
			},
			want: "test-from-event",
		},
		"use event field must keep case": {
			cfg: map[string]interface{}{"key": "Test-%{[field]}"},
			event: beat.Event{
				Fields: mapstr.M{"field": "From-Event"},
			},
			want: "Test-From-Event",
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			selector, err := buildKeySelector(common.MustNewConfigFrom(test.cfg))
			if err != nil {
				t.Fatalf("Failed to parse configuration: %v", err)
			}

			got, err := selector.Select(&test.event)
			if err != nil {
				t.Fatalf("Failed to create key name: %v", err)
			}

			if test.want != got {
				t.Errorf("Pipeline name missmatch (want: %v, got: %v)", test.want, got)
			}
		})
	}
}
