// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package action

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/ecs"
)

func TestActionFromMap(t *testing.T) {

	tests := []struct {
		Name string
		Map  map[string]interface{}
		Err  error
	}{
		{
			Name: "nil",
			Map:  nil,
			Err:  ErrActionRequest,
		},
		{
			Name: "empty",
			Map:  map[string]interface{}{},
			Err:  ErrActionRequest,
		},
		{
			Name: "invalid id",
			Map: map[string]interface{}{
				"id": 123,
			},
			Err: ErrActionRequest,
		},
		{
			Name: "invalid data",
			Map: map[string]interface{}{
				"id":   "123456789",
				"data": "foo",
			},
			Err: ErrActionRequest,
		},
		{
			Name: "invalid query",
			Map: map[string]interface{}{
				"id": "123456789",
				"data": map[string]interface{}{
					"query": 123221231,
				},
			},
			Err: ErrActionRequest,
		},
		{
			Name: "valid string for query",
			Map: map[string]interface{}{
				"id": "123456789",
				"data": map[string]interface{}{
					"query": "select * from foo",
				},
			},
		},
		{
			Name: "empty id",
			Map: map[string]interface{}{
				"id": "",
				"data": map[string]interface{}{
					"query": "select * from foo",
				},
			},
			Err: ErrActionRequest,
		},
		{
			Name: "space empty id",
			Map: map[string]interface{}{
				"id": "    ",
				"data": map[string]interface{}{
					"query": "select * from foo",
				},
			},
			Err: ErrActionRequest,
		},
		{
			Name: "empty query",
			Map: map[string]interface{}{
				"id": "123456789",
				"data": map[string]interface{}{
					"query": "",
				},
			},
			Err: ErrActionRequest,
		},
		{
			Name: "space empty query",
			Map: map[string]interface{}{
				"id": "123456789",
				"data": map[string]interface{}{
					"query": "   ",
				},
			},
			Err: ErrActionRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			a, err := FromMap(tc.Map)

			if tc.Err != nil {
				if !errors.Is(err, tc.Err) {
					t.Errorf("expected error: %v, got: %v", tc.Err, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			}

			_ = a
		})
	}
}

func TestActionFromMapWithECSMapping(t *testing.T) {

	tests := []struct {
		Name   string
		Map    map[string]interface{}
		Action Action
		Err    error
	}{
		{
			Name: "valid",
			Map: map[string]interface{}{
				"id": "214f219d-d67c-4744-8eb1-0a812594263f",
				"data": map[string]interface{}{
					"query": "select * from users limit 2",
				},
			},
			Action: Action{
				ID:    "214f219d-d67c-4744-8eb1-0a812594263f",
				Query: "select * from users limit 2",
			},
		},
		{
			Name: "ECS mapping nil",
			Map: map[string]interface{}{
				"id": "214f219d-d67c-4744-8eb1-0a812594263f",
				"data": map[string]interface{}{
					"query":       "select * from users limit 2",
					"ecs_mapping": nil,
				},
			},
			Action: Action{
				ID:    "214f219d-d67c-4744-8eb1-0a812594263f",
				Query: "select * from users limit 2",
			},
		},
		{
			Name: "ECS mapping empty string",
			Map: map[string]interface{}{
				"id": "214f219d-d67c-4744-8eb1-0a812594263f",
				"data": map[string]interface{}{
					"query":       "select * from users limit 2",
					"ecs_mapping": "",
				},
			},
			Err: ErrActionRequest,
		},
		{
			Name: "ECS mapping empty",
			Map: map[string]interface{}{
				"id": "214f219d-d67c-4744-8eb1-0a812594263f",
				"data": map[string]interface{}{
					"query":       "select * from users limit 2",
					"ecs_mapping": map[string]interface{}{},
				},
			},
			Action: Action{
				ID:         "214f219d-d67c-4744-8eb1-0a812594263f",
				Query:      "select * from users limit 2",
				ECSMapping: ecs.Mapping{},
			},
		},
		{
			Name: "ECS mapping invalid",
			Map: map[string]interface{}{
				"id": "214f219d-d67c-4744-8eb1-0a812594263f",
				"data": map[string]interface{}{
					"query": "select * from users limit 2",
					"ecs_mapping": map[string]interface{}{
						"foo": "bar",
					},
				},
			},
			Err: ErrActionRequest,
		},
		{
			Name: "ECS mapping invalid, field spaces string",
			Map: map[string]interface{}{
				"id": "214f219d-d67c-4744-8eb1-0a812594263f",
				"data": map[string]interface{}{
					"query": "select * from users limit 2",
					"ecs_mapping": map[string]interface{}{
						"user.custom.shoeSize": map[string]interface{}{
							"field": "      ",
						},
					},
				},
			},
			Err: ErrActionRequest,
		},
		{
			Name: "ECS mapping invalid, key empty",
			Map: map[string]interface{}{
				"id": "214f219d-d67c-4744-8eb1-0a812594263f",
				"data": map[string]interface{}{
					"query": "select * from users limit 2",
					"ecs_mapping": map[string]interface{}{
						"": map[string]interface{}{
							"field": "uid",
						},
					},
				},
			},
			Err: ErrActionRequest,
		},
		{
			Name: "ECS mapping invalid, key spaces",
			Map: map[string]interface{}{
				"id": "214f219d-d67c-4744-8eb1-0a812594263f",
				"data": map[string]interface{}{
					"query": "select * from users limit 2",
					"ecs_mapping": map[string]interface{}{
						"  ": map[string]interface{}{
							"field": "uid",
						},
					},
				},
			},
			Err: ErrActionRequest,
		},
		{
			Name: "ECS mapping invalid, field non-string",
			Map: map[string]interface{}{
				"id": "214f219d-d67c-4744-8eb1-0a812594263f",
				"data": map[string]interface{}{
					"query": "select * from users limit 2",
					"ecs_mapping": map[string]interface{}{
						"user.custom.shoeSize": map[string]interface{}{
							"field": 123,
						},
					},
				},
			},
			Err: ErrActionRequest,
		},
		{
			Name: "ECS mapping invalid, both field and value defined",
			Map: map[string]interface{}{
				"id": "214f219d-d67c-4744-8eb1-0a812594263f",
				"data": map[string]interface{}{
					"query": "select * from users limit 2",
					"ecs_mapping": map[string]interface{}{
						"user.custom.shoeSize": map[string]interface{}{
							"value": 48,
							"field": "uid",
						},
					},
				},
			},
			Err: ErrActionRequest,
		},
		{
			Name: "ECS mapping valid",
			Map: map[string]interface{}{
				"id": "214f219d-d67c-4744-8eb1-0a812594263f",
				"data": map[string]interface{}{
					"query": "select * from users limit 2",
					"ecs_mapping": map[string]interface{}{
						"user.custom.shoeSize": map[string]interface{}{
							"value": 48,
						},
						"user.id": map[string]interface{}{
							"field": "uid",
						},
					},
				},
			},
			Action: Action{
				ID:    "214f219d-d67c-4744-8eb1-0a812594263f",
				Query: "select * from users limit 2",
				ECSMapping: ecs.Mapping{
					"user.custom.shoeSize": ecs.MappingInfo{
						Value: int(48),
					},
					"user.id": ecs.MappingInfo{
						Field: "uid",
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			ad, err := FromMap(tc.Map)
			if tc.Err != nil {
				if !errors.Is(err, tc.Err) {
					t.Fatalf("unexpected error, want:%v, got %v", tc.Err, err)
				}
			} else {
				if err != nil {
					t.Fatal("unexpected error:", err)
				}
				diff := cmp.Diff(tc.Action, ad)
				if diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}
