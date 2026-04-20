// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package instance

import (
	"strings"

	"go.opentelemetry.io/collector/confmap"
)


// DeDotKeys converts any dot-separated keys (e.g. "path.home",
// "management.otel.enabled") into nested submaps.
func DeDotKeys(conf *confmap.Conf) error {
	nested := map[string]any{}
	for key, val := range conf.ToStringMap() {
		if !strings.Contains(key, ".") {
			continue
		}
		conf.Delete(key)
		setNested(nested, strings.Split(key, "."), val)
	}
	if len(nested) > 0 {
		return conf.Merge(confmap.NewFromStringMap(nested))
	}
	return nil
}

func setNested(m map[string]any, parts []string, val any) {
	for i, part := range parts {
		if i == len(parts)-1 {
			m[part] = val
			return
		}
		next, ok := m[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			m[part] = next
		}
		m = next
	}
}
