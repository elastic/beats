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

package elasticsearch

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
)

func TestConnectCallbacksManagement(t *testing.T) {
	f0 := func(client *eslegclient.Connection) error { fmt.Println("i am function #0"); return nil }
	f1 := func(client *eslegclient.Connection) error { fmt.Println("i am function #1"); return nil }
	f2 := func(client *eslegclient.Connection) error { fmt.Println("i am function #2"); return nil }

	_, err := RegisterConnectCallback(f0)
	if err != nil {
		t.Fatalf("error while registering callback: %v", err)
	}
	id1, err := RegisterConnectCallback(f1)
	if err != nil {
		t.Fatalf("error while registering callback: %v", err)
	}
	id2, err := RegisterConnectCallback(f2)
	if err != nil {
		t.Fatalf("error while registering callback: %v", err)
	}

	t.Logf("removing second callback")
	DeregisterConnectCallback(id1)
	if _, ok := connectCallbackRegistry.callbacks[id2]; !ok {
		t.Fatalf("third callback cannot be retrieved")
	}
}

func TestGlobalConnectCallbacksManagement(t *testing.T) {
	f0 := func(client *eslegclient.Connection) error { fmt.Println("i am function #0"); return nil }
	f1 := func(client *eslegclient.Connection) error { fmt.Println("i am function #1"); return nil }
	f2 := func(client *eslegclient.Connection) error { fmt.Println("i am function #2"); return nil }

	_, err := RegisterGlobalCallback(f0)
	if err != nil {
		t.Fatalf("error while registering callback: %v", err)
	}
	id1, err := RegisterGlobalCallback(f1)
	if err != nil {
		t.Fatalf("error while registering callback: %v", err)
	}
	id2, err := RegisterGlobalCallback(f2)
	if err != nil {
		t.Fatalf("error while registering callback: %v", err)
	}

	t.Logf("removing second callback")
	DeregisterGlobalCallback(id1)
	if _, ok := globalCallbackRegistry.callbacks[id2]; !ok {
		t.Fatalf("third callback cannot be retrieved")
	}
}

func TestPipelineSelection(t *testing.T) {
	cases := map[string]struct {
		cfg   map[string]interface{}
		event beat.Event
		want  string
	}{
		"no pipline configured": {},
		"pipeline configured": {
			cfg:  map[string]interface{}{"pipeline": "test"},
			want: "test",
		},
		"pipeline must be lowercase": {
			cfg:  map[string]interface{}{"pipeline": "Test"},
			want: "test",
		},
		"pipeline via event meta": {
			event: beat.Event{Meta: common.MapStr{"pipeline": "test"}},
			want:  "test",
		},
		"pipeline via event meta must be lowercase": {
			event: beat.Event{Meta: common.MapStr{"pipeline": "Test"}},
			want:  "test",
		},
		"pipelines setting": {
			cfg: map[string]interface{}{
				"pipelines": []map[string]interface{}{{"pipeline": "test"}},
			},
			want: "test",
		},
		"pipelines setting must be lowercase": {
			cfg: map[string]interface{}{
				"pipelines": []map[string]interface{}{{"pipeline": "Test"}},
			},
			want: "test",
		},
	}

	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			selector, err := buildPipelineSelector(common.MustNewConfigFrom(test.cfg))

			client, err := NewClient(
				ClientSettings{
					Pipeline: &selector,
				},
				nil,
			)
			assert.NoError(t, err)

			if err != nil {
				t.Fatalf("Failed to parse configuration: %v", err)
			}

			got, err := client.getPipeline(&test.event)
			if err != nil {
				t.Fatalf("Failed to create pipeline name: %v", err)
			}

			if test.want != got {
				t.Errorf("Pipeline name missmatch (want: %v, got: %v)", test.want, got)
			}
		})
	}
}
