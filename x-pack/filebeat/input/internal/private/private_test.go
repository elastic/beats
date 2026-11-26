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
	"slices"
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
		wantErr: cycle{reflect.TypeFor[map[string]any]()},
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
			wantErr: cycle{reflect.TypeFor[*s]()},
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
			wantErr: cycle{reflect.TypeFor[*s]()},
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

// maskNamed returns a MaskStrings function that masks a value if any name in
// its path is one of names, ignoring case.
func maskNamed(names ...string) func(path []string, s string) (string, bool) {
	return func(path []string, _ string) (string, bool) {
		for _, p := range path {
			for _, n := range names {
				if strings.EqualFold(p, n) {
					return "REDACTED", true
				}
			}
		}
		return "", false
	}
}

type maskTagged struct {
	Token string `json:"access_token"`
	Name  string `json:"name"`
}

var maskStringsTests = []struct {
	name  string
	in    any
	tag   string
	names []string
	want  any
}{
	{
		name:  "top_level",
		in:    map[string]any{"access_token": "a", "other": "b"},
		names: []string{"access_token"},
		want:  map[string]any{"access_token": "REDACTED", "other": "b"},
	},
	{
		name:  "case_is_decided_by_the_function",
		in:    map[string]any{"Access_Token": "a", "other": "b"},
		names: []string{"access_token"},
		want:  map[string]any{"Access_Token": "REDACTED", "other": "b"},
	},
	{
		name: "nested_at_any_depth",
		in: map[string]any{
			"data":  map[string]any{"access_token": "a", "ok": "b"},
			"items": []any{map[string]any{"access_token": "c", "ok": "d"}},
		},
		names: []string{"access_token"},
		want: map[string]any{
			"data":  map[string]any{"access_token": "REDACTED", "ok": "b"},
			"items": []any{map[string]any{"access_token": "REDACTED", "ok": "d"}},
		},
	},
	{
		name: "everything_below_a_matching_name",
		in: map[string]any{
			"password": map[string]any{"value": "a", "hint": "b"},
		},
		names: []string{"password"},
		want: map[string]any{
			"password": map[string]any{"value": "REDACTED", "hint": "REDACTED"},
		},
	},
	{
		name: "form_values",
		in: map[string][]string{
			"client_secret": {"a", "b"},
			"grant_type":    {"client_credentials"},
		},
		names: []string{"client_secret"},
		want: map[string][]string{
			"client_secret": {"REDACTED", "REDACTED"},
			"grant_type":    {"client_credentials"},
		},
	},
	{
		name:  "any_slice_with_non_strings",
		in:    map[string]any{"secret": []any{"a", true, 1.5}},
		names: []string{"secret"},
		want:  map[string]any{"secret": []any{"REDACTED", true, 1.5}},
	},
	{
		name:  "typed_string_map",
		in:    map[string]string{"api_key": "a", "other": "b"},
		names: []string{"api_key"},
		want:  map[string]string{"api_key": "REDACTED", "other": "b"},
	},
	{
		name:  "struct_field_names",
		in:    maskTagged{Token: "a", Name: "b"},
		names: []string{"Token"},
		want:  maskTagged{Token: "REDACTED", Name: "b"},
	},
	{
		name:  "struct_tag_names",
		in:    maskTagged{Token: "a", Name: "b"},
		tag:   "json",
		names: []string{"access_token"},
		want:  maskTagged{Token: "REDACTED", Name: "b"},
	},
	{
		name:  "slice_root",
		in:    []any{map[string]any{"access_token": "a", "ok": "b"}, "c"},
		names: []string{"access_token"},
		want:  []any{map[string]any{"access_token": "REDACTED", "ok": "b"}, "c"},
	},
	{
		name:  "no_match",
		in:    map[string]any{"a": "b", "c": []any{"d"}},
		names: []string{"access_token"},
		want:  map[string]any{"a": "b", "c": []any{"d"}},
	},
}

func TestMaskStrings(t *testing.T) {
	for _, test := range maskStringsTests {
		t.Run(test.name, func(t *testing.T) {
			before, err := json.Marshal(test.in)
			if err != nil {
				t.Fatalf("failed to get before state: %v", err)
			}
			got, err := RedactFunc(test.in, test.tag, MaskStrings(maskNamed(test.names...)))
			if err != nil {
				t.Fatalf("unexpected error from RedactFunc: %v", err)
			}
			after, err := json.Marshal(test.in)
			if err != nil {
				t.Fatalf("failed to get after state: %v", err)
			}
			if !bytes.Equal(before, after) {
				t.Errorf("unexpected change in input:\n---:\n+++:\n%s", cmp.Diff(before, after))
			}
			if !cmp.Equal(test.want, got) {
				t.Errorf("unexpected result:\n--- want:\n+++ got:\n%s", cmp.Diff(test.want, got))
			}
		})
	}
}

func TestMaskStrings_json_number(t *testing.T) {
	in := map[string]any{"password": json.Number("123456789"), "n": json.Number("1")}
	got, err := RedactFunc(in, "", MaskStrings(maskNamed("password")))
	if err != nil {
		t.Fatalf("unexpected error from RedactFunc: %v", err)
	}
	// The replacement must be a plain string. A json.Number holding
	// non-numeric text cannot be marshaled.
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}
	if want := `{"n":1,"password":"REDACTED"}`; string(b) != want {
		t.Errorf("unexpected result: got:%s want:%s", b, want)
	}
}

func TestRedactFunc_paths(t *testing.T) {
	in := map[string]any{
		"a": map[string]any{"b": "x"},
		"c": []any{map[string]any{"d": "y"}},
	}
	var got []string
	_, err := RedactFunc(in, "", func(v Visit) (reflect.Value, Action) {
		got = append(got, strings.Join(v.Path, "."))
		return reflect.Value{}, Keep
	})
	if err != nil {
		t.Fatalf("unexpected error from RedactFunc: %v", err)
	}
	slices.Sort(got)
	// A slice element is visited with the path of its slice.
	want := []string{"a", "a.b", "c", "c", "c.d"}
	if !cmp.Equal(want, got) {
		t.Errorf("unexpected paths:\n--- want:\n+++ got:\n%s", cmp.Diff(want, got))
	}
}

func TestRedactFunc_actions(t *testing.T) {
	fn := func(v Visit) (reflect.Value, Action) {
		switch v.Path[len(v.Path)-1] {
		case "drop", "A":
			return reflect.Value{}, Remove
		case "swap":
			return reflect.ValueOf("new"), Replace
		}
		return reflect.Value{}, Keep
	}

	t.Run("map", func(t *testing.T) {
		in := map[string]any{"drop": "a", "swap": "b", "keep": "c"}
		got, err := RedactFunc(in, "", fn)
		if err != nil {
			t.Fatalf("unexpected error from RedactFunc: %v", err)
		}
		want := map[string]any{"swap": "new", "keep": "c"}
		if !cmp.Equal(want, got) {
			t.Errorf("unexpected result:\n--- want:\n+++ got:\n%s", cmp.Diff(want, got))
		}
		if _, ok := in["drop"]; !ok {
			t.Error("input was modified")
		}
	})

	t.Run("struct", func(t *testing.T) {
		type s struct{ A, B string }
		got, err := RedactFunc(s{A: "a", B: "b"}, "", fn)
		if err != nil {
			t.Fatalf("unexpected error from RedactFunc: %v", err)
		}
		if want := (s{B: "b"}); got != want {
			t.Errorf("unexpected result: got:%+v want:%+v", got, want)
		}
	})
}

func TestRedact_slice_root_unchanged(t *testing.T) {
	in := []any{map[string]any{"private": "a", "a": "b"}}
	got, err := Redact(in, "", []string{"a"}, NewStringReplacer("REDACTED"))
	if err != nil {
		t.Fatalf("unexpected error from Redact: %v", err)
	}
	if !cmp.Equal(in, got) {
		t.Errorf("Redact changed a slice root:\n--- want:\n+++ got:\n%s", cmp.Diff(in, got))
	}
}

func TestRedactFunc_bytes_are_not_visited(t *testing.T) {
	type payload struct {
		Raw  []byte
		Name string
	}
	var calls int
	_, err := RedactFunc(payload{Raw: make([]byte, 1000), Name: "n"}, "", func(v Visit) (reflect.Value, Action) {
		calls++
		return reflect.Value{}, Keep
	})
	if err != nil {
		t.Fatalf("unexpected error from RedactFunc: %v", err)
	}
	// Only the two fields are visited, not the bytes in Raw.
	if calls != 2 {
		t.Errorf("fn was called %d times; want 2", calls)
	}
}
