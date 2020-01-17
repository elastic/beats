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

func TestConfiguration(t *testing.T) {
	beatUUID, err := uuid.NewV4()
	if err != nil {
		t.Fatalf("error while generating Beat UUID: %v", err)
	}

	server, client := newServerClientPair(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check correct path is used
		assert.Equal(t, "/api/beats/agent/"+beatUUID.String()+"/configuration", r.URL.Path)

		// Check enrollment token is correct
		assert.Equal(t, "thisismyenrollmenttoken", r.Header.Get("kbn-beats-access-token"))

		assert.Equal(t, "false", r.URL.Query().Get("validSetting"))

		fmt.Fprintf(w, `{"configuration_blocks":[{"type":"filebeat.modules","config":{"module":"apache2"}},{"type":"metricbeat.modules","config":{"module":"system","period":"10s"}}]}`)
	}))
	defer server.Close()

	configs, err := client.Configuration("thisismyenrollmenttoken", beatUUID, false)
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
			assert.Equal(t, test.equal, ConfigBlocksEqual(test.a, test.b))
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

	_, err = client.Configuration("thisismyenrollmenttoken", beatUUID, false)
	assert.True(t, IsConfigurationNotFound(err))
}
