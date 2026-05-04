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

package redact

import (
	"fmt"
	"io"
	"net/url"
	"reflect"
	"slices"
	"strings"
)

// REDACTED is the value that replaces sensitive values.
const REDACTED = "REDACTED"

// RedactOption is an optional arg to control how the Redact method works.
type RedactOption func(ro *redactOptions)

type redactOptions struct {
	errOut       io.Writer
	markerPrefix string
	ignoreKeys   []string
}

// WithErrorOutput determines where any error messages that are encountered are written to.
//
// Defaults to io.Discard.
func WithErrorOutput(w io.Writer) RedactOption {
	return func(ro *redactOptions) {
		ro.errOut = w
	}
}

// WithMarkerPrefix sets the redaction marker prefix string.
func WithMarkerPrefix(s string) RedactOption {
	return func(ro *redactOptions) {
		ro.markerPrefix = s
	}
}

// WithIgnoreKeys sets case-sensitive key values where redaction will be skipped.
func WithIgnoreKeys(keys ...string) RedactOption {
	return func(ro *redactOptions) {
		ro.ignoreKeys = append(ro.ignoreKeys, keys...)
	}
}

// Redact walks obj and replaces values of sensitive keys with a redacted
// placeholder. It mutates obj in place; nested maps and slice elements are
// modified directly and no copy is returned.
func Redact(obj map[any]any, opts ...RedactOption) {
	ro := &redactOptions{
		errOut:       io.Discard,
		markerPrefix: "",
		ignoreKeys:   []string{},
	}
	for _, opt := range opts {
		opt(ro)
	}
	redactMap(obj, ro)
}

func redactMap[K comparable](obj map[K]any, ro *redactOptions) {
	if obj == nil {
		return
	}

	markers := make([]string, 0)
	for key, val := range obj {
		// detect if the obj has entries of the form:
		// - name: Authorization
		//   value: Bearer SecretValue
		if keyString, ok := any(key).(string); ok && strings.ToLower(keyString) == "name" {
			keyVal, ok := val.(string)
			if ok && redactKey(keyVal, ro) {
				for vk := range obj {
					if vs, ok := any(vk).(string); ok && strings.ToLower(vs) == "value" {
						obj[vk] = REDACTED
						break
					}
				}
			}
		}
		if val != nil {
			switch cast := val.(type) {
			case map[string]any:
				redactMap(cast, ro)
			case map[any]any:
				redactMap(cast, ro)
			case map[int]any:
				redactMap(cast, ro)
			case []any:
				// Recursively process each element in the slice so that we also walk
				// through lists (e.g. inputs[4].streams[0]). This is required to
				// reach redaction markers that are inside slice items.
				for i, value := range cast {
					switch m := value.(type) {
					case map[string]any:
						redactMap(m, ro)
					case map[any]any:
						redactMap(m, ro)
					case map[int]any:
						redactMap(m, ro)
					case string:
						if redactedValue, redact := redactURL(m); redact {
							cast[i] = redactedValue
						}
					}
				}
			case string:
				if keyString, ok := any(key).(string); ok && redactKey(keyString, ro) {
					val = REDACTED
				} else if redactedValue, redact := redactURL(cast); redact {
					val = redactedValue
				}
			case bool: // redaction marker values are always going to be bool, process redaction markers in this case
				if keyString, ok := any(key).(string); ok {
					// Find siblings that have the redaction marker.
					if ro.markerPrefix != "" && strings.HasPrefix(keyString, ro.markerPrefix) {
						markers = append(markers, keyString)
						delete(obj, key)
						continue
					}
				}
			default:
				// in cases where we got some weird kind of map we couldn't parse, print a warning
				if reflect.TypeOf(val).Kind() == reflect.Map {
					fmt.Fprintf(ro.errOut, "[WARNING]: file may be partly redacted, could not cast value %v of type %T", key, val)
				}

			}
		}
		obj[key] = val
	}

	for _, redactionMarker := range markers {
		keyToRedact := strings.TrimPrefix(redactionMarker, ro.markerPrefix)
		for rootKey := range obj {
			if keyString, ok := any(rootKey).(string); ok {
				if keyString == keyToRedact {
					obj[rootKey] = REDACTED
				}

				if keyString == redactionMarker {
					delete(obj, rootKey)
				}
			}
		}
	}
}

func redactKey(k string, ro *redactOptions) bool {
	if slices.Contains(ro.ignoreKeys, k) {
		return false
	}

	k = strings.ToLower(k)
	return strings.Contains(k, "auth") ||
		strings.Contains(k, "certificate") ||
		strings.Contains(k, "passphrase") ||
		strings.Contains(k, "password") ||
		strings.Contains(k, "token") ||
		strings.Contains(k, "key") ||
		strings.Contains(k, "secret")
}

func redactURL(v string) (string, bool) {
	u, err := url.Parse(v)

	if err != nil || u.User == nil {
		return v, false
	}

	u.User = url.UserPassword(REDACTED, REDACTED)

	return u.String(), true
}
