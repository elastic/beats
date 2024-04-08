// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package websocket

import (
	"strings"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// redactor implements lazy field redaction of sets of a mapstr.M.
type redactor struct {
	state mapstr.M
	cfg   *redact
}

// String renders the JSON corresponding to r.state after applying redaction
// operations.
func (r redactor) String() string {
	if r.cfg == nil || len(r.cfg.Fields) == 0 {
		return r.state.String()
	}
	c := make(mapstr.M, len(r.state))
	cloneMap(c, r.state)
	for _, mask := range r.cfg.Fields {
		if r.cfg.Delete {
			walkMap(c, mask, func(parent mapstr.M, key string) {
				delete(parent, key)
			})
			continue
		}
		walkMap(c, mask, func(parent mapstr.M, key string) {
			parent[key] = "*"
		})
	}
	return c.String()
}

// cloneMap is an enhanced version of mapstr.M.Clone that handles cloning arrays
// within objects. Nested arrays are not handled.
func cloneMap(dst, src mapstr.M) {
	for k, v := range src {
		switch v := v.(type) {
		case mapstr.M:
			d := make(mapstr.M, len(v))
			dst[k] = d
			cloneMap(d, v)
		case map[string]interface{}:
			d := make(map[string]interface{}, len(v))
			dst[k] = d
			cloneMap(d, v)
		case []mapstr.M:
			a := make([]mapstr.M, 0, len(v))
			for _, m := range v {
				d := make(mapstr.M, len(m))
				cloneMap(d, m)
				a = append(a, d)
			}
			dst[k] = a
		case []map[string]interface{}:
			a := make([]map[string]interface{}, 0, len(v))
			for _, m := range v {
				d := make(map[string]interface{}, len(m))
				cloneMap(d, m)
				a = append(a, d)
			}
			dst[k] = a
		default:
			dst[k] = v
		}
	}
}

// walkMap walks to all ends of the provided path in m and applies fn to the
// final element of each walk. Nested arrays are not handled.
func walkMap(m mapstr.M, path string, fn func(parent mapstr.M, key string)) {
	key, rest, more := strings.Cut(path, ".")
	v, ok := m[key]
	if !ok {
		return
	}
	if !more {
		fn(m, key)
		return
	}
	switch v := v.(type) {
	case mapstr.M:
		walkMap(v, rest, fn)
	case map[string]interface{}:
		walkMap(v, rest, fn)
	case []mapstr.M:
		for _, m := range v {
			walkMap(m, rest, fn)
		}
	case []map[string]interface{}:
		for _, m := range v {
			walkMap(m, rest, fn)
		}
	case []interface{}:
		for _, v := range v {
			switch m := v.(type) {
			case mapstr.M:
				walkMap(m, rest, fn)
			case map[string]interface{}:
				walkMap(m, rest, fn)
			}
		}
	}
}
