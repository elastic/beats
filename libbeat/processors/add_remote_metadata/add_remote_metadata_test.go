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

package add_remote_metadata

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/elastic/beats/v7/libbeat/beat"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var addRemoteMetadataTests = []struct {
	name        string
	config      mapstr.M
	event       mapstr.M
	want        mapstr.M
	wantErr     error
	wantInitErr error
}{
	{
		name: "invalid_no_provider",
		config: mapstr.M{
			"match_keys": []string{"remote.user.id"},
			"target":     "new_field",
		},
		wantInitErr: errors.New("fail to unpack the add_remote_metadata configuration: string value is not set accessing 'provider'"),
	},
	{
		name: "invalid_no_match_keys",
		config: mapstr.M{
			"provider":       "map",
			"target":         "new_field",
			"include_fields": []string{"remote.key"},
		},
		wantInitErr: errors.New("fail to unpack the add_remote_metadata configuration: missing required field accessing 'match_keys'"),
	},
	{
		name: "invalid_no_fields_or_target",
		config: mapstr.M{
			"provider":   "map",
			"match_keys": []string{"remote.user.id"},
		},
		wantInitErr: errors.New("fail to unpack the add_remote_metadata configuration: no target field and no source fields specified accessing config"),
	},
	{
		name: "valid_scalar",
		config: mapstr.M{
			"match_keys": []string{"remote.user.id"},
			"target":     "new_field",
			"provider":   "map",
			"metadata": mapstr.M{
				"one": "the value",
				"two": "not the value",
			},
		},
		event: mapstr.M{
			"remote": mapstr.M{
				"user": mapstr.M{
					"id": "one",
				},
			},
		},
		want: mapstr.M{
			"new_field": "the value",
			"remote": mapstr.M{
				"user": mapstr.M{
					"id": "one",
				},
			},
		},
	},
	{
		name: "valid_object_root",
		config: mapstr.M{
			"match_keys":     []string{"remote.user.id"},
			"include_fields": []string{"remote.key"},
			"provider":       "map",
			"metadata": mapstr.M{
				"one": mapstr.M{"remote.key": "the value"},
				"two": mapstr.M{"remote.key": "not the value"},
			},
		},
		event: mapstr.M{
			"remote": mapstr.M{
				"user": mapstr.M{
					"id": "one",
				},
			},
		},
		want: mapstr.M{
			"remote": mapstr.M{
				"key": "the value",
				"user": mapstr.M{
					"id": "one",
				},
			},
		},
	},
	{
		name: "valid_object_non_root",
		config: mapstr.M{
			"match_keys":     []string{"remote.user.id"},
			"target":         "new_field",
			"include_fields": []string{"remote.key"},
			"provider":       "map",
			"metadata": mapstr.M{
				"one": mapstr.M{"remote.key": "the value"},
				"two": mapstr.M{"remote.key": "not the value"},
			},
		},
		event: mapstr.M{
			"remote": mapstr.M{
				"user": mapstr.M{
					"id": "one",
				},
			},
		},
		want: mapstr.M{
			"new_field": mapstr.M{
				"remote": mapstr.M{
					"key": "the value",
				},
			},
			"remote": mapstr.M{
				"user": mapstr.M{
					"id": "one",
				},
			},
		},
	},
	{
		name: "doc_people",
		config: mapstr.M{
			"match_keys":     []string{"user.name"},
			"include_fields": []string{"roles"},
			"target":         "user",
			"provider":       "map",
			"metadata": mapstr.M{
				"Alice": mapstr.M{"roles": []string{"ceo", "admin"}},
				"Bob":   mapstr.M{"roles": []string{"hr"}},
			},
		},
		event: mapstr.M{
			"user": mapstr.M{
				"name": "Alice",
			},
		},
		want: mapstr.M{
			"user": mapstr.M{
				"name":  "Alice",
				"roles": []any{"ceo", "admin"},
			},
		},
	},
}

func TestAddRemoteMetadata(t *testing.T) {
	logp.TestingSetup(logp.WithSelectors(name))
	for _, test := range addRemoteMetadataTests {
		t.Run(test.name, func(t *testing.T) {
			config, err := conf.NewConfigFrom(test.config)
			if err != nil {
				t.Fatal(err)
			}

			proc, err := New(config)
			if fmt.Sprint(err) != fmt.Sprint(test.wantInitErr) {
				t.Errorf("unexpected error from New: got:%v want:%v", err, test.wantInitErr)
			}
			if err != nil {
				return
			}
			t.Log(proc.String())
			evt := beat.Event{
				Fields: test.event,
			}
			got, err := proc.Run(&evt)
			if fmt.Sprint(err) != fmt.Sprint(test.wantErr) {
				t.Errorf("unexpected error from Run: got:%v want:%v", err, test.wantErr)
			}
			if !cmp.Equal(test.want, got.Fields) {
				t.Errorf("unexpected result\n--- want\n+++ got\n%s", cmp.Diff(test.want, got.Fields))
			}
		})
	}
}
