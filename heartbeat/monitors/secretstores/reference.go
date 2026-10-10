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

package secretstores

import (
	"fmt"
	"regexp"
	"strings"
)

// Reference points to one field of a secret: $<type>{<path>#<field>}.
type Reference struct {
	Type  string
	Path  string
	Field string
}

// String returns the reference in configuration syntax. It never contains a
// secret value, so it is safe in errors and logs.
func (r Reference) String() string {
	return "$" + r.Type + "{" + r.Path + "#" + r.Field + "}"
}

// ParseReference parses the content of a $<storeType>{...} reference.
func ParseReference(storeType, content string) (Reference, error) {
	expr := "$" + storeType + "{" + content + "}"
	if strings.Contains(content, "@") {
		return Reference{}, fmt.Errorf("%s: named connections are not supported yet in reference %s", storeType, expr)
	}

	idx := strings.LastIndex(content, "#")
	if idx < 0 {
		return Reference{}, fmt.Errorf("%s: reference %s must have the form $%s{<path>#<field>}", storeType, expr, storeType)
	}
	path := strings.Trim(content[:idx], "/")
	field := content[idx+1:]
	if path == "" || field == "" {
		return Reference{}, fmt.Errorf("%s: reference %s has an empty path or field", storeType, expr)
	}

	return Reference{Type: storeType, Path: path, Field: field}, nil
}

// referencePattern matches $<type>{<content>}. The syntax differs from ${...}
// on purpose: Elastic Agent, the OTel Collector and go-ucfg all resolve ${...}
// themselves, while they pass $<type>{...} through, so it reaches Heartbeat in
// every deployment.
//
// There is no escape: go-ucfg and the OTel Collector turn "$$" into "$" while
// Elastic Agent does not, so an escape would depend on the deployment.
var referencePattern = regexp.MustCompile(`\$([a-z][a-z0-9_]*)\{([^{}]*)\}`)

// scanReferences calls fn with the store type and content of each reference
// to a registered store type in s. Expressions with an unregistered type are
// not references.
func scanReferences(s string, fn func(storeType, content string)) {
	for _, sm := range referencePattern.FindAllStringSubmatch(s, -1) {
		if _, ok := lookupFactory(sm[1]); ok {
			fn(sm[1], sm[2])
		}
	}
}

// replaceReferences replaces the references in s with the values returned by
// resolve.
func replaceReferences(s string, resolve func(storeType, content string) (string, error)) (string, error) {
	var firstErr error
	out := referencePattern.ReplaceAllStringFunc(s, func(expr string) string {
		sm := referencePattern.FindStringSubmatch(expr)
		if _, ok := lookupFactory(sm[1]); !ok || firstErr != nil {
			return expr
		}
		v, err := resolve(sm[1], sm[2])
		if err != nil {
			firstErr = err
			return expr
		}
		return v
	})
	if firstErr != nil {
		return "", firstErr
	}
	return out, nil
}

// walkStrings calls fn on every string in v, a value unpacked from a config,
// and stores the result in place.
func walkStrings(v any, fn func(string) (string, error)) (any, error) {
	switch val := v.(type) {
	case string:
		return fn(val)
	case map[string]any:
		for k, item := range val {
			res, err := walkStrings(item, fn)
			if err != nil {
				return nil, err
			}
			val[k] = res
		}
		return val, nil
	case []any:
		for i, item := range val {
			res, err := walkStrings(item, fn)
			if err != nil {
				return nil, err
			}
			val[i] = res
		}
		return val, nil
	default:
		return v, nil
	}
}
