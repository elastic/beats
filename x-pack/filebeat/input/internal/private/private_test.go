// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package private

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type redactTest struct {
	name      string
	in        any
	tag       string
	global    []string
	replacers []Replacer
	want      any
	wantErr   error
}

var redactTests = []redactTest{
	{
		name: "map_string",
		in: map[string]any{
			"private":    "secret",
			"secret":     "1",
			"not_secret": "2",
		},
		want: map[string]any{
			"private":    "secret",
			"not_secret": "2",
		},
	},
	{
		name: "map_string_replacer",
		in: map[string]any{
			"private":    "secret",
			"secret":     "this is a secret",
			"not_secret": "this is not",
		},
		replacers: []Replacer{NewStringReplacer("REDACTED")},
		want: map[string]any{
			"private":    "secret",
			"secret":     "REDACTED",
			"not_secret": "this is not",
		},
	},
	{
		name: "map_string_custom_replacer",
		in: map[string]any{
			"private":    "secret",
			"secret":     "this is a secret",
			"not_secret": "this is not",
		},
		replacers: []Replacer{func(s string) string { return strings.Repeat("*", len(s)) }},
		want: map[string]any{
			"private":    "secret",
			"secret":     "****************", // Same length as original.
			"not_secret": "this is not",
		},
	},
	{
		name: "map_string_inner",
		in: map[string]any{
			"inner": map[string]any{
				"private":    "secret",
				"secret":     "1",
				"not_secret": "2",
			}},
		want: map[string]any{
			"inner": map[string]any{
				"private":    "secret",
				"not_secret": "2",
			}},
	},
	{
		name: "map_string_inner_global",
		in: map[string]any{
			"inner": map[string]any{
				"secret":     "1",
				"not_secret": "2",
			}},
		global: []string{"inner.secret"},
		want: map[string]any{
			"inner": map[string]any{
				"not_secret": "2",
			}},
	},
	{
		name: "map_string_inner_next_inner_global",
		in: map[string]any{
			"inner": map[string]any{
				"next_inner": map[string]any{
					"secret":     "1",
					"not_secret": "2",
				},
			}},
		global: []string{"inner.next_inner.secret"},
		want: map[string]any{
			"inner": map[string]any{
				"next_inner": map[string]any{
					"not_secret": "2",
				},
			}},
	},
	{
		name: "map_string_inner_next_inner_global_slices",
		in: map[string]any{
			"inner": map[string]any{
				"next_inner": map[string]any{
					"secret":     []string{"1"},
					"not_secret": []string{"2"},
				},
			}},
		global: []string{"inner.next_inner.secret"},
		want: map[string]any{
			"inner": map[string]any{
				"next_inner": map[string]any{
					"not_secret": []string{"2"},
				},
			}},
	},
	{
		name: "map_string_inner_next_inner_global_nested_slices",
		in: map[string]any{
			"inner": map[string]any{
				"next_inner": map[string]any{
					"secret":     [][]string{{"1"}},
					"not_secret": [][]string{{"2"}},
				},
			}},
		global: []string{"inner.next_inner.secret"},
		want: map[string]any{
			"inner": map[string]any{
				"next_inner": map[string]any{
					"not_secret": [][]string{{"2"}},
				},
			}},
	},
	{
		name: "map_string_inner_next_inner_global_slices_replacer",
		in: map[string]any{
			"inner": map[string]any{
				"next_inner": map[string]any{
					"secret":     []string{"1"},
					"not_secret": []string{"2"},
				},
			}},
		replacers: []Replacer{NewStringReplacer("REDACTED")},
		global:    []string{"inner.next_inner.secret"},
		want: map[string]any{
			"inner": map[string]any{
				"next_inner": map[string]any{
					"not_secret": []string{"2"},
					"secret":     []string{"REDACTED"},
				},
			}},
	},
	{
		name: "map_string_inner_next_inner_global_nested_slices_replacer",
		in: map[string]any{
			"inner": map[string]any{
				"next_inner": map[string]any{
					"secret":     [][]string{{"1"}},
					"not_secret": [][]string{{"2"}},
				},
			}},
		replacers: []Replacer{NewStringReplacer("REDACTED")},
		global:    []string{"inner.next_inner.secret"},
		want: map[string]any{
			"inner": map[string]any{
				"next_inner": map[string]any{
					"secret":     [][]string{{"REDACTED"}},
					"not_secret": [][]string{{"2"}},
				},
			}},
	},
	{
		name: "map_string_inner_next_inner_params_global",
		in: map[string]any{
			"inner": map[string]any{
				"next_inner": map[string]any{
					"headers": url.Values{
						"secret":     []string{"1"},
						"not_secret": []string{"2"},
					},
					"not_secret": "2",
				},
			}},
		global: []string{"inner.next_inner.headers.secret"},
		want: map[string]any{
			"inner": map[string]any{
				"next_inner": map[string]any{
					"headers": url.Values{
						"not_secret": []string{"2"},
					},
					"not_secret": "2",
				},
			}},
	},
	{
		name: "map_string_inner_next_inner_params_global_internal",
		in: map[string]any{
			"inner": map[string]any{
				"next_inner": map[string]any{
					"headers": url.Values{
						"secret":     []string{"1"},
						"not_secret": []string{"2"},
					},
					"not_secret": "2",
				},
			}},
		global: []string{"inner.next_inner.headers"},
		want: map[string]any{
			"inner": map[string]any{
				"next_inner": map[string]any{
					"not_secret": "2",
				},
			}},
	},
	{
		name: "map_string_inner_next_inner_params_global_internal_slice",
		in: map[string]any{
			"inner": map[string]any{
				"next_inner": []map[string]any{
					{
						"headers": url.Values{
							"secret":     []string{"1"},
							"not_secret": []string{"2"},
						},
						"not_secret": "2",
					},
					{
						"headers": url.Values{
							"secret":     []string{"3"},
							"not_secret": []string{"4"},
						},
						"not_secret": "4",
					},
				},
			}},
		global: []string{"inner.next_inner.headers"},
		want: map[string]any{
			"inner": map[string]any{
				"next_inner": []map[string]any{
					{"not_secret": "2"},
					{"not_secret": "4"},
				},
			}},
	},
	{
		name: "map_string_inner_next_inner_params_global_internal_slice_precise",
		in: map[string]any{
			"inner": map[string]any{
				"next_inner": []map[string]any{
					{
						"headers": url.Values{
							"secret":     []string{"1"},
							"not_secret": []string{"2"},
						},
						"not_secret": "2",
					},
					{
						"headers": url.Values{
							"secret":     []string{"3"},
							"not_secret": []string{"4"},
						},
						"not_secret": "4",
					},
				},
			}},
		global: []string{"inner.next_inner.headers.secret"},
		want: map[string]any{
			"inner": map[string]any{
				"next_inner": []map[string]any{
					{
						"headers": url.Values{
							"not_secret": []string{"2"},
						},
						"not_secret": "2",
					},
					{
						"headers": url.Values{
							"not_secret": []string{"4"},
						},
						"not_secret": "4",
					},
				},
			}},
	},
	{
		name: "map_string_inner_next_inner_params_global_internal_slice_precise_replacer",
		in: map[string]any{
			"inner": map[string]any{
				"next_inner": []map[string]any{
					{
						"headers": url.Values{
							"secret":     []string{"1"},
							"not_secret": []string{"2"},
						},
						"not_secret": "2",
					},
					{
						"headers": url.Values{
							"secret":     []string{"3"},
							"not_secret": []string{"4"},
						},
						"not_secret": "4",
					},
				},
			}},
		global:    []string{"inner.next_inner.headers.secret"},
		replacers: []Replacer{NewStringReplacer("REDACTED")},
		want: map[string]any{
			"inner": map[string]any{
				"next_inner": []map[string]any{
					{
						"headers": url.Values{
							"not_secret": []string{"2"},
							"secret":     []string{"REDACTED"},
						},
						"not_secret": "2",
					},
					{
						"headers": url.Values{
							"not_secret": []string{"4"},
							"secret":     []string{"REDACTED"},
						},
						"not_secret": "4",
					},
				},
			}},
	},
	{
		name: "map_slice",
		in: map[string]any{
			"private":    []string{"secret"},
			"secret":     "1",
			"not_secret": "2",
		},
		want: map[string]any{
			"private":    []string{"secret"},
			"not_secret": "2",
		},
	},
	{
		name: "map_cycle",
		in: func() any {
			m := map[string]any{
				"private":    "secret",
				"secret":     "1",
				"not_secret": "2",
			}
			m["loop"] = m
			return m
		}(),
		want:    map[string]any(nil),
		wantErr: cycle{reflect.TypeOf(map[string]any(nil))},
	},
	func() redactTest {
		type s struct {
			Private   string
			Secret    string
			NotSecret string
		}
		return redactTest{
			name: "struct_string",
			in: s{
				Private:   "Secret",
				Secret:    "1",
				NotSecret: "2",
			},
			tag: "",
			want: s{
				Private:   "Secret",
				NotSecret: "2",
			},
		}
	}(),
	func() redactTest {
		type s struct {
			Private   string
			Secret    string
			NotSecret string
		}
		return redactTest{
			name: "struct_string_replacer",
			in: s{
				Private:   "Secret",
				Secret:    "this is a secret",
				NotSecret: "this is not",
			},
			replacers: []Replacer{NewStringReplacer("REDACTED")},
			tag:       "",
			want: s{
				Private:   "Secret",
				Secret:    "REDACTED",
				NotSecret: "this is not",
			},
		}
	}(),
	func() redactTest {
		type s struct {
			Private   string
			Secret    string
			NotSecret string
		}
		return redactTest{
			name: "struct_string_replacer",
			in: s{
				Private:   "Secret",
				Secret:    "this is a secret",
				NotSecret: "this is not",
			},
			replacers: []Replacer{func(s string) string { return strings.Repeat("*", len(s)) }},
			tag:       "",
			want: s{
				Private:   "Secret",
				Secret:    "****************",
				NotSecret: "this is not",
			},
		}
	}(),
	func() redactTest {
		type s struct {
			Private   []string
			Secret    string
			NotSecret string
		}
		return redactTest{
			name: "struct_slice",
			in: s{
				Private:   []string{"Secret"},
				Secret:    "1",
				NotSecret: "2",
			},
			tag: "",
			want: s{
				Private:   []string{"Secret"},
				NotSecret: "2",
			},
		}
	}(),
	func() redactTest {
		type s struct {
			Private   string
			Secret    string
			NotSecret string
			Loop      *s
		}
		v := s{
			Private:   "Secret",
			Secret:    "1",
			NotSecret: "2",
		}
		v.Loop = &v
		return redactTest{
			name:    "struct_loop",
			in:      v,
			tag:     "",
			want:    s{},
			wantErr: cycle{reflect.TypeOf(&s{})},
		}
	}(),
	func() redactTest {
		type s struct {
			Private   string `json:"private"`
			Secret    string `json:"secret"`
			NotSecret string `json:"not_secret"`
		}
		return redactTest{
			name: "struct_string_json",
			in: s{
				Private:   "secret",
				Secret:    "1",
				NotSecret: "2",
			},
			tag: "json",
			want: s{
				Private:   "secret",
				NotSecret: "2",
			},
		}
	}(),
	func() redactTest {
		type s struct {
			Private   struct{} `private:"secret"`
			Secret    string   `json:"secret"`
			NotSecret string   `json:"not_secret"`
		}
		return redactTest{
			name: "struct_string_on_tag_json",
			in: s{
				Secret:    "1",
				NotSecret: "2",
			},
			tag: "json",
			want: s{
				NotSecret: "2",
			},
		}
	}(),
	func() redactTest {
		type s struct {
			Private   struct{} `private:"secret1,secret2"`
			Secret1   string   `json:"secret1"`
			Secret2   string   `json:"secret2"`
			NotSecret string   `json:"not_secret"`
		}
		return redactTest{
			name: "struct_string_list_on_tag_json",
			in: s{
				Secret1:   "1",
				Secret2:   "1",
				NotSecret: "2",
			},
			tag: "json",
			want: s{
				NotSecret: "2",
			},
		}
	}(),
	func() redactTest {
		type s struct {
			Private   string `json:"private"`
			Secret    string
			NotSecret string `json:"not_secret"`
		}
		return redactTest{
			name: "struct_string_json_missing_tag",
			in: s{
				Private:   "Secret",
				Secret:    "1",
				NotSecret: "2",
			},
			tag: "json",
			want: s{
				Private:   "Secret",
				NotSecret: "2",
			},
		}
	}(),
	func() redactTest {
		type s struct {
			Private   []string `json:"private"`
			Secret    string   `json:"secret"`
			NotSecret string   `json:"not_secret"`
		}
		return redactTest{
			name: "struct_slice_json",
			in: s{
				Private:   []string{"secret"},
				Secret:    "1",
				NotSecret: "2",
			},
			tag: "json",
			want: s{
				Private:   []string{"secret"},
				NotSecret: "2",
			},
		}
	}(),
	func() redactTest {
		type s struct {
			Private   string `json:"private"`
			Secret    string `json:"secret"`
			NotSecret string `json:"not_secret"`
			Loop      *s     `json:"loop"`
		}
		v := s{
			Private:   "secret",
			Secret:    "1",
			NotSecret: "2",
		}
		v.Loop = &v
		return redactTest{
			name:    "struct_loop_json",
			in:      v,
			tag:     "json",
			want:    s{},
			wantErr: cycle{reflect.TypeOf(&s{})},
		}
	}(),
	{
		name:      "invalid_replacer_wrong_type",
		in:        struct{}{},
		replacers: []Replacer{func(s string) int { return len(s) }},
		want:      struct{}{},
		wantErr:   errors.New("replacer does not preserve type: fn(string) int"),
	},
	{
		name:      "invalid_replacer_wrong_argnum",
		in:        struct{}{},
		replacers: []Replacer{func(a, b string) string { return a + b }},
		want:      struct{}{},
		wantErr:   errors.New("incorrect number of arguments for replacer: 2 != 1"),
	},
	{
		name:      "invalid_replacer_wrong_retnum",
		in:        struct{}{},
		replacers: []Replacer{func(s string) (a, b string) { return s, s }},
		want:      struct{}{},
		wantErr:   errors.New("incorrect number of return values from replacer: 2 != 1"),
	},
	{
		name: "invalid_replacer_collision",
		in:   struct{}{},
		replacers: []Replacer{
			func(s string) string { return s },
			func(s string) string { return s },
		},
		want:    struct{}{},
		wantErr: errors.New("multiple replacers for string"),
	},
}

func TestRedact(t *testing.T) {
	allow := cmp.AllowUnexported()

	for _, test := range redactTests {
		t.Run(test.name, func(t *testing.T) {
			var before []byte
			_, isCycle := test.wantErr.(cycle)
			if !isCycle {
				var err error
				before, err = json.Marshal(test.in)
				if err != nil {
					t.Fatalf("failed to get before state: %v", err)
				}
			}
			got, err := Redact(test.in, test.tag, test.global, test.replacers...)
			if !sameError(err, test.wantErr) {
				t.Fatalf("unexpected error from Redact: %v", err)
			}
			if err != nil {
				return
			}
			if !isCycle {
				after, err := json.Marshal(test.in)
				if err != nil {
					t.Fatalf("failed to get after state: %v", err)
				}
				if !bytes.Equal(before, after) {
					t.Errorf("unexpected change in input:\n---:\n+++:\n%s", cmp.Diff(before, after))
				}
			}
			if !cmp.Equal(test.want, got, allow) {
				t.Errorf("unexpected paths:\n--- want:\n+++ got:\n%s", cmp.Diff(test.want, got, allow))
			}
		})
	}
}

func sameError(a, b error) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil, b == nil:
		return false
	default:
		return a.Error() == b.Error()
	}
}
