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

package add_kubernetes_metadata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors/actions"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Test Annotator is skipped if kubernetes metadata already exist
func TestAnnotatorSkipped(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"lookup_fields": []string{"kubernetes.pod.name"},
	})
	matcher, err := NewFieldMatcher(*cfg)
	if err != nil {
		t.Fatal(err)
	}

	processor := kubernetesAnnotator{
		log:   logp.NewLogger(selector),
		cache: newCache(10 * time.Second),
		matchers: &Matchers{
			matchers: []Matcher{matcher},
		},
		kubernetesAvailable: true,
	}

	processor.cache.set("foo",
		mapstr.M{
			"kubernetes": mapstr.M{
				"pod": mapstr.M{
					"labels": mapstr.M{
						"added": "should not",
					},
				},
			},
		})

	event, err := processor.Run(&beat.Event{
		Fields: mapstr.M{
			"kubernetes": mapstr.M{
				"pod": mapstr.M{
					"name": "foo",
					"id":   "pod_id",
					"metrics": mapstr.M{
						"a": 1,
						"b": 2,
					},
				},
			},
		},
	})
	assert.NoError(t, err)

	assert.Equal(t, mapstr.M{
		"kubernetes": mapstr.M{
			"pod": mapstr.M{
				"name": "foo",
				"id":   "pod_id",
				"metrics": mapstr.M{
					"a": 1,
					"b": 2,
				},
			},
		},
	}, event.Fields)
}

// Test metadata are not included in the event
func TestAnnotatorWithNoKubernetesAvailable(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"lookup_fields": []string{"kubernetes.pod.name"},
	})
	matcher, err := NewFieldMatcher(*cfg)
	if err != nil {
		t.Fatal(err)
	}

	processor := kubernetesAnnotator{
		cache: newCache(10 * time.Second),
		matchers: &Matchers{
			matchers: []Matcher{matcher},
		},
		kubernetesAvailable: false,
	}

	intialEventMap := mapstr.M{
		"kubernetes": mapstr.M{
			"pod": mapstr.M{
				"name": "foo",
				"id":   "pod_id",
				"metrics": mapstr.M{
					"a": 1,
					"b": 2,
				},
			},
		},
	}

	event, err := processor.Run(&beat.Event{
		Fields: intialEventMap.Clone(),
	})
	assert.NoError(t, err)

	assert.Equal(t, intialEventMap, event.Fields)
}

// Test field is being added correctly based on indexer and matcher
func TestAddField(t *testing.T) {
	cfg := config.MustNewConfigFrom(map[string]interface{}{
		"matchers": []map[string]interface{}{
			{
				"field_format": map[string]interface{}{
					"format": "%{[destination.ip]}:%{[destination.port]}",
				},
			},
		},
		"add_fields": map[string]interface{}{
			"target": "flow",
			"fields": map[string]interface{}{
				"destination": "destination",
			},
		},
	})

	processorConfig, err := newProcessorConfig(cfg, NewRegister())
	if err != nil {
		t.Fatal(err)
	}
	matcher := NewMatchers(processorConfig.Matchers)

	annotatorCache := newCache(10 * time.Second)
	// add an indexer that we want to test
	// Since the test is for ip_port, add ip:port as the key
	annotatorCache.set("192.16.2.13:53", mapstr.M{})

	addFieldProcessor, err := actions.CreateAddFields(processorConfig.AddFields)
	if err != nil {
		t.Fatal(err)
	}

	processor := kubernetesAnnotator{
		log:   logp.NewLogger(selector).With("libbeat.processor", "add_kubernetes_metadata_test"),
		cache: annotatorCache,
		matchers: &Matchers{
			matchers: []Matcher{matcher},
		},
		kubernetesAvailable: true,
		addFieldProcessor:   addFieldProcessor,
	}

	intialEventMap := mapstr.M{
		"testMap": "test",
		"destination": mapstr.M{
			"port":    53,
			"bytes":   408,
			"ip":      "192.16.2.13",
			"packets": 2,
		},
	}

	expectedEventMap := mapstr.M{
		"testMap": "test",
		"destination": mapstr.M{
			"port":    53,
			"bytes":   408,
			"ip":      "192.16.2.13",
			"packets": 2,
		},
		"flow": mapstr.M{
			"destination": "destination",
		},
	}

	event, err := processor.Run(&beat.Event{
		Fields: intialEventMap.Clone(),
	})
	assert.NoError(t, err)

	assert.Equal(t, expectedEventMap, event.Fields)
}

// TestNewProcessorConfigDefaultIndexers validates the behaviour of default indexers and
// matchers settings
func TestNewProcessorConfigDefaultIndexers(t *testing.T) {
	emptyRegister := NewRegister()
	registerWithDefaults := NewRegister()
	registerWithDefaults.AddDefaultIndexerConfig("ip_port", *config.NewConfig())
	registerWithDefaults.AddDefaultMatcherConfig("field_format", *config.MustNewConfigFrom(map[string]interface{}{
		"format": "%{[destination.ip]}:%{[destination.port]}",
	}))

	configWithIndexersAndMatchers := config.MustNewConfigFrom(map[string]interface{}{
		"indexers": []map[string]interface{}{
			{
				"container": map[string]interface{}{},
			},
		},
		"matchers": []map[string]interface{}{
			{
				"fields": map[string]interface{}{
					"lookup_fields": []string{"container.id"},
				},
			},
		},
	})
	configOverrideDefaults := config.MustNewConfigFrom(map[string]interface{}{
		"default_indexers.enabled": "false",
		"default_matchers.enabled": "false",
	})
	require.NoError(t, configOverrideDefaults.Merge(configWithIndexersAndMatchers))

	cases := map[string]struct {
		register         *Register
		config           *config.C
		expectedMatchers []string
		expectedIndexers []string
	}{
		"no matchers": {
			register: emptyRegister,
			config:   config.NewConfig(),
		},
		"one configured indexer and matcher": {
			register:         emptyRegister,
			config:           configWithIndexersAndMatchers,
			expectedIndexers: []string{"container"},
			expectedMatchers: []string{"fields"},
		},
		"default indexers and matchers": {
			register:         registerWithDefaults,
			config:           config.NewConfig(),
			expectedIndexers: []string{"ip_port"},
			expectedMatchers: []string{"field_format"},
		},
		"default indexers and matchers, don't use indexers": {
			register: registerWithDefaults,
			config: config.MustNewConfigFrom(map[string]interface{}{
				"default_indexers.enabled": "false",
			}),
			expectedMatchers: []string{"field_format"},
		},
		"default indexers and matchers, don't use matchers": {
			register: registerWithDefaults,
			config: config.MustNewConfigFrom(map[string]interface{}{
				"default_matchers.enabled": "false",
			}),
			expectedIndexers: []string{"ip_port"},
		},
		"one configured indexer and matcher and defaults, configured should come first": {
			register:         registerWithDefaults,
			config:           configWithIndexersAndMatchers,
			expectedIndexers: []string{"container", "ip_port"},
			expectedMatchers: []string{"fields", "field_format"},
		},
		"override defaults": {
			register:         registerWithDefaults,
			config:           configOverrideDefaults,
			expectedIndexers: []string{"container"},
			expectedMatchers: []string{"fields"},
		},
	}

	names := func(plugins PluginConfig) []string {
		var ns []string
		for _, plugin := range plugins {
			for name := range plugin {
				ns = append(ns, name)
			}
		}
		return ns
	}

	for title, c := range cases {
		t.Run(title, func(t *testing.T) {
			config, err := newProcessorConfig(c.config, c.register)
			require.NoError(t, err)
			assert.Equal(t, c.expectedMatchers, names(config.Matchers), "expected matchers")
			assert.Equal(t, c.expectedIndexers, names(config.Indexers), "expected indexers")
		})
	}
}
