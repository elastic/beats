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
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

const redactedURL = REDACTED + ":" + REDACTED

func TestRedact(t *testing.T) {
	const markerPrefix = "__mark_redact_"

	tests := []struct {
		name   string
		input  map[string]any
		opts   []RedactOption
		expect map[string]any
	}{
		{
			name:   "nil map",
			input:  nil,
			expect: nil,
		},
		{
			name:   "empty map",
			input:  map[string]any{},
			expect: map[string]any{},
		},
		{
			name: "no redactions",
			input: map[string]any{
				"type":      "elasticsearch",
				"namespace": "default",
				"count":     int64(5),
			},
			expect: map[string]any{
				"type":      "elasticsearch",
				"namespace": "default",
				"count":     int64(5),
			},
		},
		{
			name: "sensitive keys are redacted",
			input: map[string]any{
				"api_key":     "secret",
				"password":    "secret",
				"passphrase":  "secret",
				"token":       "secret",
				"certificate": "secret",
				"secret":      "secret",
				"X-App-Auth":  "secret",
				"safe":        "value",
			},
			expect: map[string]any{
				"api_key":     REDACTED,
				"password":    REDACTED,
				"passphrase":  REDACTED,
				"token":       REDACTED,
				"certificate": REDACTED,
				"secret":      REDACTED,
				"X-App-Auth":  REDACTED,
				"safe":        "value",
			},
		},
		{
			name: "keys are matched case insensitively",
			input: map[string]any{
				"API_KEY":     "secret",
				"PassWord":    "secret",
				"PASSPHRASE":  "secret",
				"tOkEn":       "secret",
				"Certificate": "secret",
			},
			expect: map[string]any{
				"API_KEY":     REDACTED,
				"PassWord":    REDACTED,
				"PASSPHRASE":  REDACTED,
				"tOkEn":       REDACTED,
				"Certificate": REDACTED,
			},
		},
		{
			name: "URL credentials are redacted",
			//nolint:gosec // this test is meant to redact sensitive values
			input: map[string]any{
				"url":       "https://user:pass@example.com/path",
				"other_url": "https://example.com/path",
				"plain":     "not a url",
			},
			expect: map[string]any{
				"url":       "https://" + redactedURL + "@example.com/path",
				"other_url": "https://example.com/path",
				"plain":     "not a url",
			},
		},
		{
			name: "URL credentials in slice elements are redacted",
			input: map[string]any{
				"urls": []any{
					"https://user:pass@my-url1",
					"https://user:pass@my-url2",
					"https://my-url3",
				},
			},
			expect: map[string]any{
				"urls": []any{
					"https://" + redactedURL + "@my-url1",
					"https://" + redactedURL + "@my-url2",
					"https://my-url3",
				},
			},
		},
		{
			name: "sensitive key wins over URL redaction",
			//nolint:gosec // this test is meant to redact sensitive values
			input: map[string]any{
				"secret_url": "https://user:pass@example.com",
			},
			expect: map[string]any{
				"secret_url": REDACTED,
			},
		},
		{
			name: "nested map[string]any is redacted recursively",
			input: map[string]any{
				"outputs": map[string]any{
					"default": map[string]any{
						"type":     "elasticsearch",
						"api_key":  "secret",
						"hosts":    []any{"https://user:pass@es:9200"},
						"username": "user",
					},
				},
			},
			expect: map[string]any{
				"outputs": map[string]any{
					"default": map[string]any{
						"type":     "elasticsearch",
						"api_key":  REDACTED,
						"hosts":    []any{"https://" + redactedURL + "@es:9200"},
						"username": "user",
					},
				},
			},
		},
		{
			name: "nested map[any]any is redacted recursively",
			input: map[string]any{
				"outputs": map[any]any{
					"default": map[any]any{
						"type":    "elasticsearch",
						"api_key": "secret",
					},
				},
			},
			expect: map[string]any{
				"outputs": map[any]any{
					"default": map[any]any{
						"type":    "elasticsearch",
						"api_key": REDACTED,
					},
				},
			},
		},
		{
			name: "slice items that are maps are redacted recursively",
			input: map[string]any{
				"inputs": []any{
					map[string]any{
						"type":    "test",
						"api_key": "secret",
					},
					map[string]any{
						"type":     "test",
						"password": "secret",
					},
				},
			},
			expect: map[string]any{
				"inputs": []any{
					map[string]any{
						"type":    "test",
						"api_key": REDACTED,
					},
					map[string]any{
						"type":     "test",
						"password": REDACTED,
					},
				},
			},
		},
		{
			name: "deeply nested ssl key in inputs is redacted",
			input: map[string]any{
				"inputs": []any{
					map[string]any{
						"ssl": map[string]any{
							"certificate": "cert1",
							"key":         "key1",
						},
						"nested": map[string]any{
							"ssl": map[string]any{
								"certificate": "cert2",
								"key":         "key2",
							},
						},
						"slice": []any{
							map[string]any{
								"ssl": map[string]any{
									"certificate": "cert3",
									"key":         "key3",
								},
							},
						},
					},
				},
			},
			expect: map[string]any{
				"inputs": []any{
					map[string]any{
						"ssl": map[string]any{
							"certificate": REDACTED,
							"key":         REDACTED,
						},
						"nested": map[string]any{
							"ssl": map[string]any{
								"certificate": REDACTED,
								"key":         REDACTED,
							},
						},
						"slice": []any{
							map[string]any{
								"ssl": map[string]any{
									"certificate": REDACTED,
									"key":         REDACTED,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "name/value entries are redacted when name indicates a secret",
			input: map[string]any{
				"headers": []any{
					map[string]any{
						"name":  "Authorization",
						"value": "Bearer secret-token",
					},
					map[string]any{
						"name":  "X-Custom",
						"value": "harmless",
					},
				},
			},
			expect: map[string]any{
				"headers": []any{
					map[string]any{
						"name":  "Authorization",
						"value": REDACTED,
					},
					map[string]any{
						"name":  "X-Custom",
						"value": "harmless",
					},
				},
			},
		},
		{
			name: "redaction markers are processed when prefix is configured",
			input: map[string]any{
				"inputs": []any{
					map[string]any{
						"type":                     "test_input",
						"redactKey":                "secretValue",
						markerPrefix + "redactKey": true,
					},
				},
				"outputs": map[string]any{
					"default": map[string]any{
						"type":                          "elasticsearch",
						"api_key":                       "alreadyMatched",
						"redactOtherKey":                "secretOutputValue",
						markerPrefix + "redactOtherKey": true,
					},
				},
			},
			opts: []RedactOption{WithMarkerPrefix(markerPrefix)},
			expect: map[string]any{
				"inputs": []any{
					map[string]any{
						"type":      "test_input",
						"redactKey": REDACTED,
					},
				},
				"outputs": map[string]any{
					"default": map[string]any{
						"type":           "elasticsearch",
						"api_key":        REDACTED,
						"redactOtherKey": REDACTED,
					},
				},
			},
		},
		{
			name: "redaction markers in nested slice items are processed",
			input: map[string]any{
				"id": "test-policy",
				"inputs": []any{
					map[string]any{
						"type": "httpjson",
						"streams": []any{
							map[string]any{
								"request": map[string]any{
									"transforms": []any{
										map[string]any{
											"set": map[string]any{
												"target":               "header.Authorization",
												"value":                "SSWS this-should-be-redacted",
												markerPrefix + "value": true,
											},
										},
										map[string]any{
											"set": map[string]any{
												"target": "url.params.limit",
												"value":  "1000",
											},
										},
									},
								},
							},
							map[string]any{
								"mock_stream_config": map[string]any{
									"kind": map[string]any{
										"string_value": "mock_stream_config_name",
									},
								},
								markerPrefix + "mock_stream_config": true,
							},
						},
					},
				},
			},
			opts: []RedactOption{WithMarkerPrefix(markerPrefix)},
			expect: map[string]any{
				"id": "test-policy",
				"inputs": []any{
					map[string]any{
						"type": "httpjson",
						"streams": []any{
							map[string]any{
								"request": map[string]any{
									"transforms": []any{
										map[string]any{
											"set": map[string]any{
												"target": "header.Authorization",
												"value":  REDACTED,
											},
										},
										map[string]any{
											"set": map[string]any{
												"target": "url.params.limit",
												"value":  "1000",
											},
										},
									},
								},
							},
							map[string]any{
								"mock_stream_config": REDACTED,
							},
						},
					},
				},
			},
		},
		{
			name: "redaction marker key is deleted from the map",
			// Explicitly verifies that the bool branch's delete (followed by
			// continue) removes the marker key so it does not survive in the
			// output. Without the continue, the trailing obj[key] = val would
			// re-add the marker before the post-loop cleanup.
			input: map[string]any{
				"redactKey":                "secretValue",
				markerPrefix + "redactKey": true,
			},
			opts: []RedactOption{WithMarkerPrefix(markerPrefix)},
			expect: map[string]any{
				"redactKey": REDACTED,
			},
		},
		{
			name: "redaction marker without matching target is still deleted",
			// The marker points to a key that does not exist in the map; the
			// marker itself must still be removed from the output.
			input: map[string]any{
				"safe":                   "value",
				markerPrefix + "missing": true,
			},
			opts: []RedactOption{WithMarkerPrefix(markerPrefix)},
			expect: map[string]any{
				"safe": "value",
			},
		},
		{
			name: "redaction markers are ignored when no prefix is configured",
			input: map[string]any{
				"benign":                "value",
				markerPrefix + "benign": true,
			},
			expect: map[string]any{
				"benign":                "value",
				markerPrefix + "benign": true,
			},
		},
		{
			name: "ignored keys are not redacted",
			input: map[string]any{
				"routekey":      "should-not-redact",
				"my_secret_key": "redact-me",
				"safe":          "value",
			},
			opts: []RedactOption{WithIgnoreKeys("routekey")},
			expect: map[string]any{
				"routekey":      "should-not-redact",
				"my_secret_key": REDACTED,
				"safe":          "value",
			},
		},
		{
			name: "ignored keys and markers coexist",
			input: map[string]any{
				"routekey":              "should-not-redact",
				"keepme":                "value",
				markerPrefix + "keepme": true,
				"api_key":               "secret",
			},
			opts: []RedactOption{
				WithIgnoreKeys("routekey"),
				WithMarkerPrefix(markerPrefix),
			},
			expect: map[string]any{
				"routekey": "should-not-redact",
				"keepme":   REDACTED,
				"api_key":  REDACTED,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var errOut bytes.Buffer
			opts := append([]RedactOption{WithErrorOutput(&errOut)}, tc.opts...)

			Redact(tc.input, opts...)

			assert.Equal(t, tc.expect, tc.input)
			assert.Empty(t, errOut.String(), "no warnings expected")
		})
	}
}

func TestRedactURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expect   string
		redacted bool
	}{
		{
			name:     "plain string",
			input:    "not a url",
			expect:   "not a url",
			redacted: false,
		},
		{
			name:     "url without credentials",
			input:    "https://example.com/path",
			expect:   "https://example.com/path",
			redacted: false,
		},
		//nolint:gosec // this test is meant to redact sensitive values
		{
			name:     "url with credentials",
			input:    "https://user:pass@example.com/path",
			expect:   "https://" + redactedURL + "@example.com/path",
			redacted: true,
		},
		{
			name:     "url with username only",
			input:    "https://user@example.com",
			expect:   "https://" + redactedURL + "@example.com",
			redacted: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, redacted := redactURL(tc.input)
			assert.Equal(t, tc.expect, got)
			assert.Equal(t, tc.redacted, redacted)
		})
	}
}

func TestRedactKey(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		ignoreKeys []string
		expect     bool
	}{
		{name: "empty", key: "", expect: false},
		{name: "safe", key: "type", expect: false},
		{name: "auth substring", key: "X-Authentication", expect: true},
		{name: "certificate", key: "certificate", expect: true},
		{name: "passphrase", key: "passphrase", expect: true},
		{name: "password", key: "password", expect: true},
		{name: "token", key: "token", expect: true},
		{name: "key substring", key: "api_key", expect: true},
		{name: "secret substring", key: "client_secret", expect: true},
		{name: "uppercase matches", key: "PASSWORD", expect: true},
		{name: "mixed case matches", key: "ApiKey", expect: true},
		{name: "ignored exact", key: "routekey", ignoreKeys: []string{"routekey"}, expect: false},
		{
			// "RouteKey" still matches the redaction rule via "key";
			// ignoreKeys check is exact-match before lowercasing.
			name: "ignored is case sensitive", key: "RouteKey", ignoreKeys: []string{"routekey"}, expect: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ro := &redactOptions{ignoreKeys: tc.ignoreKeys}
			assert.Equal(t, tc.expect, redactKey(tc.key, ro))
		})
	}
}
