// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package beater

import (
	"errors"
	"testing"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/internal/ecs"
	"github.com/google/go-cmp/cmp"
)

func TestActionDataFromRequest(t *testing.T) {

	tests := []struct {
		Name       string
		Req        map[string]interface{}
		ActionData actionData
		Err        error
	}{
		{
			Name: "nil",
			Req:  nil,
			Err:  ErrActionRequest,
		},
		{
			Name: "empty",
			Req:  map[string]interface{}{},
			Err:  ErrActionRequest,
		},
		{
			Name: "valid",
			Req: map[string]interface{}{
				"id": "214f219d-d67c-4744-8eb1-0a812594263f",
				"data": map[string]interface{}{
					"query": "select * from users limit 2",
				},
			},
			ActionData: actionData{
				ID:    "214f219d-d67c-4744-8eb1-0a812594263f",
				Query: "select * from users limit 2",
			},
		},
		{
			Name: "ECS mapping nil",
			Req: map[string]interface{}{
				"id": "214f219d-d67c-4744-8eb1-0a812594263f",
				"data": map[string]interface{}{
					"query":       "select * from users limit 2",
					"ecs_mapping": nil,
				},
			},
			Err: ErrActionRequest,
		},
		{
			Name: "ECS mapping empty string",
			Req: map[string]interface{}{
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
			Req: map[string]interface{}{
				"id": "214f219d-d67c-4744-8eb1-0a812594263f",
				"data": map[string]interface{}{
					"query":       "select * from users limit 2",
					"ecs_mapping": map[string]interface{}{},
				},
			},
			ActionData: actionData{
				ID:         "214f219d-d67c-4744-8eb1-0a812594263f",
				Query:      "select * from users limit 2",
				ECSMapping: ecs.Mapping{},
			},
		},
		{
			Name: "ECS mapping invalid",
			Req: map[string]interface{}{
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
			Req: map[string]interface{}{
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
			Req: map[string]interface{}{
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
			Req: map[string]interface{}{
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
			Req: map[string]interface{}{
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
			Req: map[string]interface{}{
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
			Req: map[string]interface{}{
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
			ActionData: actionData{
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
			ad, err := actionDataFromRequest(tc.Req)
			if tc.Err != nil {
				if !errors.Is(err, tc.Err) {
					t.Fatalf("unexpected error, want:%v, got %v", tc.Err, err)
				}
			} else {
				if err != nil {
					t.Fatal("unexpected error:", err)
				}
				diff := cmp.Diff(tc.ActionData, ad)
				if diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}
