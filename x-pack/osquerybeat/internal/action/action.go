// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package action

import (
	"errors"
	"fmt"
	"strings"

	"github.com/elastic/beats/v8/x-pack/osquerybeat/internal/ecs"
)

var (
	ErrActionRequest = errors.New("invalid action request")
)

type Action struct {
	Query      string
	ID         string
	ECSMapping ecs.Mapping
}

func FromMap(m map[string]interface{}) (a Action, err error) {
	if len(m) == 0 {
		return a, ErrActionRequest
	}

	var (
		id, query string
	)

	if v, ok := m["id"]; ok {
		if id, ok = v.(string); !ok {
			return a, fmt.Errorf("invalid id: %w", ErrActionRequest)
		}
	}

	var ecsm ecs.Mapping
	if v, ok := m["data"]; ok {
		var data map[string]interface{}
		if data, ok = v.(map[string]interface{}); !ok {
			return a, fmt.Errorf("invalid data: %w", ErrActionRequest)
		}

		if v, ok = data["query"]; ok {
			if query, ok = v.(string); !ok {
				return a, fmt.Errorf("invalid query: %w", ErrActionRequest)
			}
		}
		// Parse optional ECS Mapping
		if v, ok := data["ecs_mapping"]; ok && v != nil {
			m, ok := v.(map[string]interface{})
			if !ok {
				return a, fmt.Errorf("invalid ECS mapping: %w", ErrActionRequest)
			}
			ecsm, err = parseECSMapping(m)
			if err != nil {
				return a, err
			}
		}
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return a, fmt.Errorf("missing id: %w", ErrActionRequest)
	}

	query = strings.TrimSpace(query)
	if query == "" {
		return a, fmt.Errorf("missing query: %w", ErrActionRequest)
	}

	return Action{
		Query:      query,
		ID:         id,
		ECSMapping: ecsm,
	}, nil
}

func parseECSMapping(m map[string]interface{}) (ecsm ecs.Mapping, err error) {
	ecsm = make(ecs.Mapping)
	for k, v := range m {
		k = strings.TrimSpace(k)
		if k == "" {
			return ecsm, ErrActionRequest
		}
		valmap, ok := v.(map[string]interface{})
		if !ok {
			return ecsm, ErrActionRequest
		}

		var (
			val   interface{}
			field string
		)

		if val, ok = valmap["field"]; ok {
			// "field" can only be string
			field, ok = val.(string)
			if !ok {
				return ecsm, ErrActionRequest
			}
			field = strings.TrimSpace(field)
		}

		value := valmap["value"]
		if field != "" && value != nil {
			return ecsm, ErrActionRequest
		}

		// Should have at field or value defined in the mapping object
		if field == "" && value == nil {
			return ecsm, ErrActionRequest
		}
		ecsm[k] = ecs.MappingInfo{
			Field: field,
			Value: value,
		}
	}
	return
}
