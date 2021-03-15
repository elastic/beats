// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleetapi

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestActionSerialization(t *testing.T) {
	a := ActionApp{
		ActionID:   "1231232",
		ActionType: "APP_INPUT",
		InputType:  "osquery",
		Data:       []byte(`{ "foo": "bar" }`),
	}

	m, err := a.MarshalMap()
	if err != nil {
		t.Fatal(err)
	}

	diff := cmp.Diff(4, len(m))
	if diff != "" {
		t.Error(diff)
	}

	diff = cmp.Diff(a.ActionID, mapStringVal(m, "id"))
	if diff != "" {
		t.Error(diff)
	}

	diff = cmp.Diff(a.ActionType, mapStringVal(m, "type"))
	if diff != "" {
		t.Error(diff)
	}

	diff = cmp.Diff(a.InputType, mapStringVal(m, "input_type"))
	if diff != "" {
		t.Error(diff)
	}

	diff = cmp.Diff([]byte(a.Data), mapBytesVal(m, "data"))
	if diff != "" {
		t.Error(diff)
	}

	diff = cmp.Diff(a.StartedAt, mapStringVal(m, "started_at"))
	if diff != "" {
		t.Error(diff)
	}
	diff = cmp.Diff(a.CompletedAt, mapStringVal(m, "completed_at"))
	if diff != "" {
		t.Error(diff)
	}
	diff = cmp.Diff(a.Error, mapStringVal(m, "error"))
	if diff != "" {
		t.Error(diff)
	}
}

func mapStringVal(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func mapBytesVal(m map[string]interface{}, key string) []byte {
	if v, ok := m[key]; ok {
		if b, ok := v.([]interface{}); ok {
			res := make([]byte, 0, len(b))
			for _, v := range b {
				if val, ok := v.(byte); ok {
					res = append(res, val)
				}
			}
			return res
		}
	}
	return nil
}
