// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package streaming

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

var urlTests = []struct {
	name string
	src  string
	want any
}{
	{
		name: "parse_url",
		src:  `"http://example.com/".parse_url()`,
		want: map[string]any{
			"ForceQuery":  false,
			"Fragment":    "",
			"Host":        "example.com",
			"Opaque":      "",
			"Path":        "/",
			"RawFragment": "",
			"RawPath":     "",
			"RawQuery":    "",
			"Scheme":      "http",
			"User":        nil,
		},
	},
	{
		name: "format_url",
		src:  `{"url": {"Host": "example.com", "Path": "/", "Scheme": "https"}.format_url()}`,
		want: map[string]any{"url": "https://example.com/"},
	},
	{
		name: "parse_query",
		src:  `"q=1&a=42".parse_query()`,
		want: map[string]any{"a": []any{"42"}, "q": []any{"1"}},
	},
	{
		name: "format_query",
		src:  `{"query": {"q": ["1"], "a": ["42"]}.format_query()}`,
		want: map[string]any{"query": "a=42&q=1"},
	},
}

func TestUrlLib(t *testing.T) {
	now := time.Date(2009, 11, 10, 23, 0, 0, 0, time.UTC)
	ctx := context.Background()
	for _, test := range urlTests {
		t.Run(test.name, func(t *testing.T) {
			prg, ast, err := newProgram(ctx, test.src, "state", nil, logptest.NewTestingLogger(t, ""))
			if err != nil {
				t.Fatalf("failed to compile src: %v", err)
			}
			got, err := evalWith(ctx, prg, ast, map[string]any{}, now)
			if err != nil {
				t.Fatalf("failed to run program: %v", err)
			}
			if !cmp.Equal(test.want, got) {
				t.Errorf("unexpected result\n--- want\n+++ got\n%s", cmp.Diff(test.want, got))
			}
		})
	}
}
