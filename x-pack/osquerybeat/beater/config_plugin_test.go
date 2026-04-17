// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/config"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/ecs"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/testutil"
	"github.com/elastic/elastic-agent-libs/logp"
)

func renderFullConfigJSON(inputs []config.InputConfig) (string, error) {
	packs := make(map[string]config.Pack)
	for _, input := range inputs {
		pack := config.Pack{
			Platform:  input.Platform,
			Version:   input.Version,
			Discovery: input.Discovery,
			Queries:   make(map[string]config.Query),
		}
		for _, stream := range input.Streams {
			snapshot := true
			query := config.Query{
				Query:          stream.Query,
				NativeSchedule: config.NativeSchedule{Interval: stream.Interval},
				Platform:       stream.Platform,
				Version:        stream.Version,
				ECSMapping:     stream.ECSMapping,
				Snapshot:       &snapshot, // enforce snapshot for all queries
			}
			pack.Queries[stream.ID] = query
		}
		packs[input.Name] = pack
	}
	raw, err := newOsqueryConfig(&config.OsqueryConfig{
		Packs: packs,
	}).Render()
	if err != nil {
		return "", err
	}

	return string(raw), nil
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

func generateTestMapping(depth int, k string, v interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	res := m
	key := 'a'

	for i := 0; i < depth; i++ {
		newmap := make(map[string]interface{})
		m[string(key)] = newmap
		m = newmap
		key += 1
	}
	m[k] = v
	return res
}

func TestFlattenECSMappingEdges(t *testing.T) {

	// zero depth map should return ErrECSMappingIsInvalid
	m := generateTestMapping(0, keyValue, 1)
	_, err := flattenECSMapping(m)
	if !errors.Is(err, ErrECSMappingIsInvalid) {
		t.Fatalf("want error: %v, got: %v", ErrECSMappingIsInvalid, err)
	}

	m = generateTestMapping(0, keyField, "foo")
	_, err = flattenECSMapping(m)
	if !errors.Is(err, ErrECSMappingIsInvalid) {
		t.Fatalf("want error: %v, got: %v", ErrECSMappingIsInvalid, err)
	}

	// max depth key map should flatten
	m = generateTestMapping(maxECSMappingDepth, "value", 1)
	_, err = flattenECSMapping(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// max + 1 depth key map should return error
	m = generateTestMapping(maxECSMappingDepth+1, "value", 2)
	_, err = flattenECSMapping(m)
	if !errors.Is(err, ErrECSMappingIsTooDeep) {
		t.Fatalf("expected error: %v", ErrECSMappingIsTooDeep)
	}
}

func TestFlattenECSMappingMoreEdges(t *testing.T) {

	keys := map[string]string{
		"empty key":             "",
		"key with whitespaces":  "   ",
		"key with escaped dots": "foo\\.bar",
	}

	values := map[string]struct {
		m   interface{}
		err error
	}{
		"empty field": {
			map[string]interface{}{
				"field": "",
			},
			ErrECSMappingIsInvalid,
		},
		"empty field with whitespaces": {
			map[string]interface{}{
				"field": "   ",
			},
			ErrECSMappingIsInvalid,
		},
		"nil field": {
			map[string]interface{}{
				"field": nil,
			},
			ErrECSMappingIsInvalid,
		},
		"empty string value": {
			map[string]interface{}{
				"value": "",
			},
			nil,
		},
		"empty string value with whitespaces": {
			map[string]interface{}{
				"value": "   ",
			},
			nil,
		},
		"nil value": {
			map[string]interface{}{
				"value": nil,
			},
			nil,
		},
	}

	for depth := 1; depth < maxECSMappingDepth; depth++ {
		for keyname, key := range keys {
			for valname, val := range values {
				name := keyname + " " + valname
				t.Run(name, func(t *testing.T) {
					m := generateTestMapping(depth, key, val.m)
					_, err := flattenECSMapping(m)

					expectErr := val.err
					if strings.TrimSpace(key) == "" {
						expectErr = ErrECSMappingIsInvalid
					}
					if !errors.Is(err, expectErr) {
						t.Fatalf("want error: %v, got: %v", expectErr, err)
					}
				})
			}
		}
	}
}

func TestSet(t *testing.T) {
	logger := logp.NewLogger("config_test")

	const noQueriesConfig = `{
    "options": {
        "schedule_max_drift": 60,
        "schedule_splay_percent": 10
    }
}`
	oneInputConfig := []config.InputConfig{
		{
			Name: "osquery-manager-1",
			Type: "osquery",
			Datastream: config.DatastreamConfig{
				Namespace: "custom",
			},
			Platform: "posix",
			Version:  "4.7.0",
			Discovery: []string{
				"SELECT pid FROM processes WHERE name = 'foobar';",
				"SELECT 1 FROM users WHERE username like 'www%';",
			},
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
	}

	oneInputPackConfig, err := renderFullConfigJSON(oneInputConfig)
	if err != nil {
		t.Fatal(err)
	}

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
			name:   "one input",
			inputs: oneInputConfig,
			scfg:   oneInputPackConfig,
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

			// Should not resolve the query until the config was generated
			if tc.name == "one input" {
				_, ok := cfgp.LookupQueryInfo("users")
				diff = cmp.Diff(false, ok)
				if diff != "" {
					t.Fatal(diff)
				}

				// Check the namespaces set before configuration is generated
				for _, input := range tc.inputs {
					_, ok := cfgp.LookupNamespace("users")
					diff = cmp.Diff(false, ok)
					if diff != "" {
						t.Fatal(diff)
					}

					diff = cmp.Diff(oneInputConfig[0].Datastream.Namespace, input.Datastream.Namespace)
					if diff != "" {
						t.Fatal(diff)
					}

				}
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
					name := getPackQueryName(input.Name, stream.ID)

					ns, ok := cfgp.LookupNamespace(name)
					if !ok {
						t.Fatalf("failed to resolve namespace for %v", name)
					}

					qi, ok := cfgp.LookupQueryInfo(name)
					if !ok {
						t.Fatalf("failed to resolve name %v", name)
					}
					diff = cmp.Diff(qi.Query, stream.Query)
					if diff != "" {
						t.Error(diff)
					}

					diff = cmp.Diff(input.Datastream.Namespace, ns)
					if diff != "" {
						t.Error(diff)
					}

					if len(stream.ECSMapping) == 0 {
						continue
					}

					diff = cmp.Diff(tc.ecsm, qi.ECSMapping)
					if diff != "" {
						t.Error(diff)
					}
				}
			}

			// test that unknown query can't be resolved
			_, ok = cfgp.LookupQueryInfo("unknown query name")
			if ok {
				t.Fatalf("unexpectedly resolved unknown query")
			}
		})
	}
}

func TestSetScheduledQueryProfileFlag(t *testing.T) {
	logger := logp.NewLogger("config_test")
	cfgp := NewConfigPlugin(logger)

	inputs := []config.InputConfig{
		{
			Name: "osquery-manager-1",
			Type: "osquery",
			Datastream: config.DatastreamConfig{
				Namespace: "custom",
			},
			Osquery: &config.OsqueryConfig{
				Schedule: map[string]config.Query{
					"scheduled_users": {
						Query: "select * from users limit 1",
						NativeSchedule: config.NativeSchedule{
							Interval: 60,
						},
						Profile: true,
					},
				},
			},
		},
	}

	if err := cfgp.Set(inputs); err != nil {
		t.Fatal(err)
	}
	if _, err := cfgp.GenerateConfig(context.Background()); err != nil {
		t.Fatal(err)
	}

	if !cfgp.LookupQueryProfile("scheduled_users") {
		t.Fatal("expected scheduled query profile flag to be enabled")
	}
}

func TestSet_ScheduleMetadataIncludesSpaceID(t *testing.T) {
	logger := logp.NewLogger("config_test")
	cfgp := NewConfigPlugin(logger)

	const (
		queryName   = "scheduled_query"
		scheduleID  = "sched-123"
		startDate   = "2026-02-01T00:00:00Z"
		spaceID     = "space-abc"
		querySQL    = "select * from uptime"
		queryPeriod = 300
	)

	inputs := []config.InputConfig{
		{
			Name: "osquery-manager-1",
			Type: "osquery",
			Datastream: config.DatastreamConfig{
				Namespace: "custom",
			},
			Osquery: &config.OsqueryConfig{
				Schedule: map[string]config.Query{
					queryName: {
						Query: querySQL,
						NativeSchedule: config.NativeSchedule{
							Interval:   queryPeriod,
							ScheduleID: scheduleID,
							StartDate:  startDate,
						},
						SpaceID: spaceID,
					},
				},
			},
		},
	}

	if err := cfgp.Set(inputs); err != nil {
		t.Fatal(err)
	}

	// Query metadata becomes active after GenerateConfig.
	cfg, err := cfgp.GenerateConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	rendered, ok := cfg[configName]
	if !ok {
		t.Fatalf("missing %v configuration name", configName)
	}

	if !strings.Contains(rendered, `"space_id": "space-abc"`) {
		t.Fatalf("rendered config missing space_id: %s", rendered)
	}

	qi, ok := cfgp.LookupQueryInfo(queryName)
	if !ok {
		t.Fatalf("failed to resolve query info for %s", queryName)
	}

	if diff := cmp.Diff(querySQL, qi.Query); diff != "" {
		t.Error(diff)
	}
	if diff := cmp.Diff(scheduleID, qi.ScheduleID); diff != "" {
		t.Error(diff)
	}
	if diff := cmp.Diff(startDate, qi.StartDate); diff != "" {
		t.Error(diff)
	}
	if diff := cmp.Diff(spaceID, qi.SpaceID); diff != "" {
		t.Error(diff)
	}
	if diff := cmp.Diff(queryPeriod, qi.Interval); diff != "" {
		t.Error(diff)
	}
}

func TestSet_ScheduleMetadataIncludesPackID(t *testing.T) {
	logger := logp.NewLogger("config_test")
	cfgp := NewConfigPlugin(logger)

	const (
		packName    = "my-pack"
		packID      = "pack-uuid-123"
		queryName   = "uptime_query"
		querySQL    = "select * from uptime"
		queryPeriod = 60
	)

	inputs := []config.InputConfig{
		{
			Name: "osquery-manager-1",
			Type: "osquery",
			Datastream: config.DatastreamConfig{
				Namespace: "default",
			},
			Osquery: &config.OsqueryConfig{
				Packs: map[string]config.Pack{
					packName: {
						PackID: packID,
						Queries: map[string]config.Query{
							queryName: {
								Query: querySQL,
								NativeSchedule: config.NativeSchedule{
									Interval: queryPeriod,
								},
							},
						},
					},
				},
			},
		},
	}

	if err := cfgp.Set(inputs); err != nil {
		t.Fatal(err)
	}

	_, err := cfgp.GenerateConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	packQueryName := getPackQueryName(packName, queryName)
	qi, ok := cfgp.LookupQueryInfo(packQueryName)
	if !ok {
		t.Fatalf("failed to resolve query info for %s", packQueryName)
	}

	if diff := cmp.Diff(packID, qi.PackID); diff != "" {
		t.Error(diff)
	}
}

func TestSet_PackNativeMetadataWithoutIntervalRejected(t *testing.T) {
	logger := logp.NewLogger("config_test")
	cfgp := NewConfigPlugin(logger)
	inputs := []config.InputConfig{
		{
			Name:       "osquery-manager-1",
			Type:       "osquery",
			Datastream: config.DatastreamConfig{Namespace: "default"},
			Osquery: &config.OsqueryConfig{
				Packs: map[string]config.Pack{
					"bad": {
						DefaultNativeSchedule: config.NativeSchedule{ScheduleID: "only-id"},
						Queries:               map[string]config.Query{"q": {Query: "select 1", NativeSchedule: config.NativeSchedule{Interval: 60}}},
					},
				},
			},
		},
	}
	err := cfgp.Set(inputs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, config.ErrPackNativeScheduleMetadataWithoutInterval) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSet_PackDefaultNativeScheduleMergedIntoQueries(t *testing.T) {
	logger := logp.NewLogger("config_test")
	cfgp := NewConfigPlugin(logger)

	const (
		packName  = "infra"
		queryName = "uptime"
	)
	inputs := []config.InputConfig{
		{
			Name: "osquery-manager-1",
			Type: "osquery",
			Datastream: config.DatastreamConfig{
				Namespace: "default",
			},
			Osquery: &config.OsqueryConfig{
				Packs: map[string]config.Pack{
					packName: {
						DefaultNativeSchedule: config.NativeSchedule{
							Interval:   300,
							ScheduleID: "pack-default-sched",
							StartDate:  "2026-01-01T00:00:00Z",
						},
						Queries: map[string]config.Query{
							queryName: {Query: "select * from uptime"},
						},
					},
				},
			},
		},
	}

	if err := cfgp.Set(inputs); err != nil {
		t.Fatal(err)
	}
	if _, err := cfgp.GenerateConfig(context.Background()); err != nil {
		t.Fatal(err)
	}

	fullName := getPackQueryName(packName, queryName)
	qi, ok := cfgp.LookupQueryInfo(fullName)
	if !ok {
		t.Fatalf("lookup %s", fullName)
	}
	if diff := cmp.Diff(300, qi.Interval); diff != "" {
		t.Error(diff)
	}
	if diff := cmp.Diff("pack-default-sched", qi.ScheduleID); diff != "" {
		t.Error(diff)
	}
	if diff := cmp.Diff("2026-01-01T00:00:00Z", qi.StartDate); diff != "" {
		t.Error(diff)
	}
}

func TestSet_PackConflictingScheduleDefaultsRejected(t *testing.T) {
	logger := logp.NewLogger("config_test")
	cfgp := NewConfigPlugin(logger)
	inputs := []config.InputConfig{
		{
			Name:       "osquery-manager-1",
			Type:       "osquery",
			Datastream: config.DatastreamConfig{Namespace: "default"},
			Osquery: &config.OsqueryConfig{
				Packs: map[string]config.Pack{
					"bad": {
						DefaultNativeSchedule: config.NativeSchedule{Interval: 60},
						DefaultRRuleSchedule: &config.RRuleScheduleConfig{
							RRule:     "FREQ=DAILY",
							StartDate: "2024-01-01T00:00:00Z",
						},
						Queries: map[string]config.Query{"q": {Query: "select 1"}},
					},
				},
			},
		},
	}
	err := cfgp.Set(inputs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, config.ErrPackConflictingScheduleDefaults) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSet_PackMixedQueryScheduleModesRejected(t *testing.T) {
	logger := logp.NewLogger("config_test")
	cfgp := NewConfigPlugin(logger)
	inputs := []config.InputConfig{
		{
			Name:       "osquery-manager-1",
			Type:       "osquery",
			Datastream: config.DatastreamConfig{Namespace: "default"},
			Osquery: &config.OsqueryConfig{
				Packs: map[string]config.Pack{
					"mixed": {
						Queries: map[string]config.Query{
							"native": {Query: "select 1", NativeSchedule: config.NativeSchedule{Interval: 60}},
							"idle":   {Query: "select 2"},
						},
					},
				},
			},
		},
	}
	err := cfgp.Set(inputs)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, config.ErrPackMixedScheduleModes) {
		t.Fatalf("unexpected error: %v", err)
	}
}
