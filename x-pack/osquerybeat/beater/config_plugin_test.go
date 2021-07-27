// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/ecs"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/testutil"
	"github.com/google/go-cmp/cmp"
)

func renderFullConfig(inputs []config.InputConfig) (map[string]string, error) {
	packs := make(map[string]pack)
	for _, input := range inputs {
		pack := pack{
			Queries: make(map[string]query),
		}
		for _, stream := range input.Streams {
			query := query{
				Query:    stream.Query,
				Interval: stream.Interval,
				Platform: stream.Platform,
				Version:  stream.Version,
				Snapshot: true, // enforce snapshot for all queries
			}
			pack.Queries[stream.ID] = query
		}
		packs[input.Name] = pack
	}
	raw, err := newOsqueryConfig(packs).render()
	if err != nil {
		return nil, err
	}

	return map[string]string{
		configName: string(raw),
	}, nil
}

func TestConfigPluginNew(t *testing.T) {
	validLogger := logp.NewLogger("config_test")

	tests := []struct {
		name        string
		log         *logp.Logger
		dataPath    string
		shouldPanic bool
	}{
		{
			name:        "invalid",
			log:         nil,
			dataPath:    "",
			shouldPanic: true,
		},
		{
			name:     "empty",
			log:      validLogger,
			dataPath: "",
		},
		{
			name:     "nonempty",
			log:      validLogger,
			dataPath: "data",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.shouldPanic {
				testutil.AssertPanic(t, func() { NewConfigPlugin(tc.log) })
				return
			}

			p := NewConfigPlugin(tc.log)
			if p == nil {
				t.Fatal("nil config plugin")
			}
		})
	}
}

func TestFlattenECSMapping(t *testing.T) {
	const mapping = `{"user":{"custom":{"shoeSize":{"value":45}},"id":{"field":"uid"},"name":{"field":"username"}}}`
	var m map[string]interface{}
	err := json.Unmarshal([]byte(mapping), &m)
	if err != nil {
		t.Fatal(err)
	}
	ecsm, err := flattenECSMapping(m)
	if err != nil {
		t.Fatal(err)
	}

	diff := cmp.Diff(ecsm["user.custom.shoeSize"].Value, float64(45))
	if diff != "" {
		t.Error(diff)
	}

	diff = cmp.Diff(ecsm["user.id"].Field, "uid")
	if diff != "" {
		t.Error(diff)
	}

	diff = cmp.Diff(ecsm["user.name"].Field, "username")
	if diff != "" {
		t.Error(diff)
	}
}

func TestFlattenECSMappingTooDeep(t *testing.T) {
	const mapping = `{"a":{"b":{"c":{"d":{"e":{"f":{"g":{"h":{"i":{"j":{"k":{"value": 1}}}}}}}}}}}}`
	var m map[string]interface{}
	err := json.Unmarshal([]byte(mapping), &m)
	if err != nil {
		t.Fatal(err)
	}
	_, err = flattenECSMapping(m)
	if err != ErrECSMappingIsTooDeep {
		t.Fatalf("expected error: %v", ErrECSMappingIsTooDeep)
	}
}

func TestSet(t *testing.T) {
	logger := logp.NewLogger("config_test")

	const noQueriesConfig = `{
    "options": {
        "schedule_splay_percent": 10
    }
}`

	const oneInputConfig = `{
    "options": {
        "schedule_splay_percent": 10
    },
    "packs": {
        "osquery-manager-1": {
            "queries": {
                "users": {
                    "query": "select * from users limit 2",
                    "interval": 60,
                    "snapshot": true
                }
            }
        }
    }
}`

	tests := []struct {
		name   string
		inputs []config.InputConfig
		err    error
		scfg   string
		ecsm   ecs.Mapping
	}{
		{
			name: "nil",
			scfg: noQueriesConfig,
		},
		{
			name:   "empty",
			inputs: []config.InputConfig{},
			scfg:   noQueriesConfig,
		},
		{
			name: "one input",
			inputs: []config.InputConfig{
				{
					Name: "osquery-manager-1",
					Type: "osquery",
					Streams: []config.StreamConfig{
						{
							ID:       "users",
							Query:    "select * from users limit 2",
							Interval: 60,
							ECSMapping: map[string]interface{}{
								"user": map[string]interface{}{
									"custom": map[string]interface{}{
										"shoeSize": map[string]interface{}{
											"value": 45,
										},
									},
									"id": map[string]interface{}{
										"field": "uid",
									},
									"name": map[string]interface{}{
										"field": "username",
									},
								},
							},
						},
					},
				},
			},
			scfg: oneInputConfig,
			ecsm: ecs.Mapping{
				"user.custom.shoeSize": ecs.MappingInfo{
					Value: 45,
				},
				"user.id": ecs.MappingInfo{
					Field: "uid",
				},
				"user.name": ecs.MappingInfo{
					Field: "username",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfgp := NewConfigPlugin(logger)

			err := cfgp.Set(tc.inputs)
			diff := cmp.Diff(tc.err, err)
			if diff != "" {
				t.Fatal(diff)
			}

			// test generate config
			mcfg, err := cfgp.GenerateConfig(context.Background())
			if err != nil {
				t.Fatal(err)
			}

			scfg, ok := mcfg[configName]
			if !ok {
				t.Errorf("missing %v configuration name", configName)
			}

			diff = cmp.Diff(tc.scfg, scfg)
			if diff != "" {
				fmt.Println(scfg)
				t.Error(diff)
			}

			// test the count matches the number of inputs
			diff = cmp.Diff(len(tc.inputs), cfgp.Count())
			if diff != "" {
				t.Error(diff)
			}

			// test that the queries can be resolved
			for _, input := range tc.inputs {
				for _, stream := range input.Streams {
					name := strings.Join([]string{"pack", input.Name, stream.ID}, "_")
					sql, ok := cfgp.ResolveName(name)
					if !ok {
						t.Fatalf("failed to resolve name %v", name)
					}
					diff = cmp.Diff(sql, stream.Query)
					if diff != "" {
						t.Error(diff)
					}
					if len(stream.ECSMapping) == 0 {
						continue
					}

					// test that the query ecs mapping lookup succeeds
					ecsm, ok := cfgp.LookupECSMapping(name)
					if !ok {
						t.Fatalf("failed to lookup ecs mapping for %v", name)
					}
					diff = cmp.Diff(tc.ecsm, ecsm)
				}
			}

			// test that unknown query can't be resolved
			_, ok = cfgp.ResolveName("unknown query name")
			if ok {
				t.Fatalf("unexpectedly resolved unknown query")
			}
		})
	}
}
