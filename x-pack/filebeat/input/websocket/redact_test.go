// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package websocket

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

var redactorTests = []struct {
	name  string
	state mapstr.M
	cfg   *redact

	wantOrig   string
	wantRedact string
}{
	{
		name: "nil_redact",
		state: mapstr.M{
			"auth": mapstr.M{
				"user": "fred",
				"pass": "top_secret",
			},
			"other": "data",
		},
		cfg:        nil,
		wantOrig:   `{"auth":{"pass":"top_secret","user":"fred"},"other":"data"}`,
		wantRedact: `{"auth":{"pass":"top_secret","user":"fred"},"other":"data"}`,
	},
	{
		name: "auth_no_delete",
		state: mapstr.M{
			"auth": mapstr.M{
				"user": "fred",
				"pass": "top_secret",
			},
			"other": "data",
		},
		cfg: &redact{
			Fields: []string{"auth"},
			Delete: false,
		},
		wantOrig:   `{"auth":{"pass":"top_secret","user":"fred"},"other":"data"}`,
		wantRedact: `{"auth":"*","other":"data"}`,
	},
	{
		name: "auth_delete",
		state: mapstr.M{
			"auth": mapstr.M{
				"user": "fred",
				"pass": "top_secret",
			},
			"other": "data",
		},
		cfg: &redact{
			Fields: []string{"auth"},
			Delete: true,
		},
		wantOrig:   `{"auth":{"pass":"top_secret","user":"fred"},"other":"data"}`,
		wantRedact: `{"other":"data"}`,
	},
	{
		name: "pass_no_delete",
		state: mapstr.M{
			"auth": mapstr.M{
				"user": "fred",
				"pass": "top_secret",
			},
			"other": "data",
		},
		cfg: &redact{
			Fields: []string{"auth.pass"},
			Delete: false,
		},
		wantOrig:   `{"auth":{"pass":"top_secret","user":"fred"},"other":"data"}`,
		wantRedact: `{"auth":{"pass":"*","user":"fred"},"other":"data"}`,
	},
	{
		name: "pass_delete",
		state: mapstr.M{
			"auth": mapstr.M{
				"user": "fred",
				"pass": "top_secret",
			},
			"other": "data",
		},
		cfg: &redact{
			Fields: []string{"auth.pass"},
			Delete: true,
		},
		wantOrig:   `{"auth":{"pass":"top_secret","user":"fred"},"other":"data"}`,
		wantRedact: `{"auth":{"user":"fred"},"other":"data"}`,
	},
	{
		name: "multi_cursor_no_delete",
		state: mapstr.M{
			"cursor": []mapstr.M{
				{"key": "val_one", "other": "data"},
				{"key": "val_two", "other": "data"},
			},
			"other": "data",
		},
		cfg: &redact{
			Fields: []string{"cursor.key"},
			Delete: false,
		},
		wantOrig:   `{"cursor":[{"key":"val_one","other":"data"},{"key":"val_two","other":"data"}],"other":"data"}`,
		wantRedact: `{"cursor":[{"key":"*","other":"data"},{"key":"*","other":"data"}],"other":"data"}`,
	},
	{
		name: "multi_cursor_delete",
		state: mapstr.M{
			"cursor": []mapstr.M{
				{"key": "val_one", "other": "data"},
				{"key": "val_two", "other": "data"},
			},
			"other": "data",
		},
		cfg: &redact{
			Fields: []string{"cursor.key"},
			Delete: true,
		},
		wantOrig:   `{"cursor":[{"key":"val_one","other":"data"},{"key":"val_two","other":"data"}],"other":"data"}`,
		wantRedact: `{"cursor":[{"other":"data"},{"other":"data"}],"other":"data"}`,
	},
}

func TestRedactor(t *testing.T) {
	for _, test := range redactorTests {
		t.Run(test.name, func(t *testing.T) {
			got := fmt.Sprint(redactor{state: test.state, cfg: test.cfg})
			orig := fmt.Sprint(test.state)
			if orig != test.wantOrig {
				t.Errorf("unexpected original state after redaction:\n--- got\n--- want\n%s", cmp.Diff(orig, test.wantOrig))
			}
			if got != test.wantRedact {
				t.Errorf("unexpected redaction:\n--- got\n--- want\n%s", cmp.Diff(got, test.wantRedact))
			}
		})
	}
}
