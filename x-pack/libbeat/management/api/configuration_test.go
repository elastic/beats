// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
)

// {"list":[{"id":"6c385a04-f315-489e-9208-c87f41911782","type":"filebeat.inputs","config":{"paths":["/tmp/hello.log"]},"tag":"89be4cfd-6249-4ac2-abe2-8f82520ba435"},{"id":"315ff7e9-ae24-4c99-a9d0-ed4314bc8e60","type":"output","config":{"_sub_type":"elasticsearch","username":"elastic","password":"changeme"},"tag":"89be4cfd-6249-4ac2-abe2-8f82520ba435"}],"success":true}
func TestConfiguration(t *testing.T) {
	beatUUID, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("error while generating Beat ID: %v", err)
	}

	server, client := newServerClientPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check correct path is used
		assert.Equal(t, "/api/beats/agent/"+beatUUID.String()+"/configuration", r.URL.Path)

		// Check enrollment token is correct
		assert.Equal(t, "thisismyenrollmenttoken", r.Header.Get("kbn-beats-access-token"))

		fmt.Fprintf(w, `{"success": true, "list":[{"type":"filebeat.modules","config":{"_sub_type":"apache2"}},{"type":"metricbeat.modules","config":{"_sub_type":"system","period":"10s"}}]}`)
	}))
	defer server.Close()

	auth := AuthClient{Client: client, AccessToken: "thisismyenrollmenttoken", BeatUUID: beatUUID}

	configs, err := auth.Configuration()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 2, len(configs))
	checked := 0
	for _, config := range configs {
		if config.Type == "metricbeat.modules" {
			assert.Equal(t, &ConfigBlock{Raw: map[string]interface{}{
				"module": "system",
				"period": "10s",
			}}, config.Blocks[0])
			checked++

		} else if config.Type == "filebeat.modules" {
			assert.Equal(t, &ConfigBlock{Raw: map[string]interface{}{
				"module": "apache2",
			}}, config.Blocks[0])
			checked++
		}
	}

	assert.Equal(t, 2, checked)
}

func TestConfigBlocksEqual(t *testing.T) {
	tests := []struct {
		name  string
		a, b  ConfigBlocks
		equal bool
	}{
		{
			name:  "empty lists or nil",
			a:     nil,
			b:     ConfigBlocks{},
			equal: true,
		},
		{
			name: "single element",
			a: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "metricbeat.modules",
					Blocks: []*ConfigBlock{
						&ConfigBlock{
							Raw: map[string]interface{}{
								"foo": "bar",
							},
						},
					},
				},
			},
			b: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "metricbeat.modules",
					Blocks: []*ConfigBlock{
						&ConfigBlock{
							Raw: map[string]interface{}{
								"foo": "bar",
							},
						},
					},
				},
			},
			equal: true,
		},
		{
			name: "single element with slices",
			a: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "metricbeat.modules",
					Blocks: []*ConfigBlock{
						&ConfigBlock{
							Raw: map[string]interface{}{
								"foo": []string{"foo", "bar"},
							},
						},
					},
				},
			},
			b: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "metricbeat.modules",
					Blocks: []*ConfigBlock{
						&ConfigBlock{
							Raw: map[string]interface{}{
								"foo": []string{"foo", "bar"},
							},
						},
					},
				},
			},
			equal: true,
		},
		{
			name: "different number of blocks",
			a: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "metricbeat.modules",
					Blocks: []*ConfigBlock{
						&ConfigBlock{
							Raw: map[string]interface{}{
								"foo": "bar",
							},
						},
						&ConfigBlock{
							Raw: map[string]interface{}{
								"baz": "buzz",
							},
						},
					},
				},
			},
			b: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "metricbeat.modules",
					Blocks: []*ConfigBlock{
						&ConfigBlock{
							Raw: map[string]interface{}{
								"foo": "bar",
							},
						},
					},
				},
			},
			equal: false,
		},
		{
			name: "different block",
			a: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "metricbeat.modules",
					Blocks: []*ConfigBlock{
						&ConfigBlock{
							Raw: map[string]interface{}{
								"baz": "buzz",
							},
						},
					},
				},
			},
			b: ConfigBlocks{
				ConfigBlocksWithType{
					Type: "metricbeat.modules",
					Blocks: []*ConfigBlock{
						&ConfigBlock{
							Raw: map[string]interface{}{
								"foo": "bar",
							},
						},
					},
				},
			},
			equal: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			check, err := ConfigBlocksEqual(test.a, test.b)
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, test.equal, check)
		})
	}
}

func TestUnEnroll(t *testing.T) {
	beatUUID, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("error while generating Beat UUID: %v", err)
	}

	server, client := newServerClientPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check correct path is used
		assert.Equal(t, "/api/beats/agent/"+beatUUID.String()+"/configuration", r.URL.Path)

		// Check enrollment token is correct
		assert.Equal(t, "thisismyenrollmenttoken", r.Header.Get("kbn-beats-access-token"))

		http.NotFound(w, r)
	}))
	defer server.Close()

	auth := AuthClient{Client: client, AccessToken: "thisismyenrollmenttoken", BeatUUID: beatUUID}
	_, err = auth.Configuration()
	assert.True(t, IsConfigurationNotFound(err))
}
