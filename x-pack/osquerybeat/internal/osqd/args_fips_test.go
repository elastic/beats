// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build requirefips

package osqd

import (
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
)

// wantFIPSFlags, wantFIPSOptions and wantFIPSTables are spelled out here, not
// taken from fipsFlags and fipsDisabledTables, so that removing an entry fails
// the tests.
var (
	wantFIPSFlags = []string{
		"--disable_distributed=true",
		"--disable_enrollment=true",
		"--disable_carver=true",
		"--aws_enforce_fips=true",
		"--yara_sigurl_authenticate=false",
	}

	wantFIPSOptions = map[string]any{
		"disable_distributed":      true,
		"disable_enrollment":       true,
		"disable_carver":           true,
		"aws_enforce_fips":         true,
		"yara_sigurl_authenticate": false,
	}

	wantFIPSTables = []string{
		"carves",
		"curl",
		"curl_certificate",
		"prometheus_metrics",
		"ec2_instance_metadata",
		"ec2_instance_tags",
		"azure_instance_metadata",
		"azure_instance_tags",
		"ycloud_instance_metadata",
		"yara_file",
		"yara_process",
		"yara_events",
		"rpm_packages",
		"rpm_package_files",
	}
)

func TestArgsFIPSEnforcedFlags(t *testing.T) {
	tests := []struct {
		name  string
		flags Flags
	}{
		{name: "no_user_flags"},
		{
			name: "user_turns_remote_features_on",
			flags: Flags{
				"disable_distributed":      false,
				"disable_enrollment":       false,
				"disable_carver":           false,
				"aws_enforce_fips":         false,
				"yara_sigurl_authenticate": true,
			},
		},
		{
			name: "user_flags_as_strings",
			flags: Flags{
				"disable_distributed":      "false",
				"yara_sigurl_authenticate": "true",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			osq, err := newOsqueryD("/var/run/foobar", WithLogger(logptest.NewTestingLogger(t, "osqueryd_test")))
			require.NoError(t, err, "creating osqueryd")

			args := osq.args(tc.flags)

			for _, want := range wantFIPSFlags {
				assert.Contains(t, args, want, "FIPS builds must pass %s", want)
			}
		})
	}
}

func TestArgsFIPSWarnsAboutReplacedFlags(t *testing.T) {
	tests := []struct {
		name  string
		flags Flags
		want  int
	}{
		{name: "no_user_flags"},
		{name: "same_value", flags: Flags{"disable_distributed": true}},
		{name: "same_value_as_string", flags: Flags{"disable_distributed": "true"}},
		{name: "different_value", flags: Flags{"disable_distributed": false}, want: 1},
		{
			name: "two_different_values",
			flags: Flags{
				"disable_distributed":      false,
				"yara_sigurl_authenticate": true,
			},
			want: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			log, logs := logptest.NewTestingLoggerWithObserver(t, "osqueryd_test")
			osq, err := newOsqueryD("/var/run/foobar", WithLogger(log))
			require.NoError(t, err, "creating osqueryd")

			osq.args(tc.flags)

			got := logs.FilterMessageSnippet("Ignoring osqueryd flag").Len()
			assert.Equal(t, tc.want, got, "warnings for flags %v", tc.flags)
		})
	}
}

func TestArgsFIPSDisabledTables(t *testing.T) {
	tests := []struct {
		name  string
		flags Flags
		// extra are tables expected in addition to the FIPS tables.
		extra []string
	}{
		{name: "no_user_flags"},
		{
			name:  "user_enables_blocked_tables",
			flags: Flags{flagEnableTables: "curl,rpm_packages,yara_file"},
		},
		{
			name:  "user_disables_other_tables",
			flags: Flags{flagDisableTables: "processes, users"},
			extra: []string{"processes", "users"},
		},
		{
			name: "user_enables_and_disables",
			flags: Flags{
				flagEnableTables:  "curl,carves",
				flagDisableTables: "processes",
			},
			extra: []string{"processes"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			osq, err := newOsqueryD("/var/run/foobar", WithLogger(logptest.NewTestingLogger(t, "osqueryd_test")))
			require.NoError(t, err, "creating osqueryd")

			args := osq.args(tc.flags)

			// The default disabled tables are part of the expected set, so that
			// adding a default does not break this test.
			want := toSet(wantFIPSTables, defaultDisabledTables, tc.extra)
			got := disabledTablesFromArgs(args)
			assert.ElementsMatch(t, want, got, "unexpected disabled tables in %v", args)
		})
	}
}

func TestEnforceFIPSOptions(t *testing.T) {
	tests := []struct {
		name    string
		options map[string]any
		// others are options expected to pass through unchanged.
		others map[string]any
	}{
		{name: "nil_options"},
		{name: "empty_options", options: map[string]any{}},
		{
			name: "user_turns_remote_features_on",
			options: map[string]any{
				"disable_distributed":      false,
				"disable_enrollment":       false,
				"disable_carver":           false,
				"aws_enforce_fips":         false,
				"yara_sigurl_authenticate": true,
			},
		},
		{
			name: "other_options_pass_through",
			options: map[string]any{
				"disable_distributed":    false,
				"schedule_splay_percent": 5,
				"logger_min_status":      1,
			},
			others: map[string]any{
				"schedule_splay_percent": 5,
				"logger_min_status":      1,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Keep a copy to check that the caller's map is not changed.
			before := maps.Clone(tc.options)

			got := EnforceFIPSOptions(tc.options)

			want := make(map[string]any)
			maps.Copy(want, wantFIPSOptions)
			maps.Copy(want, tc.others)
			assert.Equal(t, want, got, "enforced options for %v", tc.options)

			assert.Equal(t, before, tc.options, "the caller's options must not change")
		})
	}
}

func TestMergeTables(t *testing.T) {
	tests := []struct {
		name  string
		list  string
		names []string
		want  string
	}{
		{name: "both_empty"},
		{name: "list_only", list: "b,a", want: "a,b"},
		{name: "names_only", names: []string{"b", "a"}, want: "a,b"},
		{name: "duplicates_and_spaces", list: " a , b,,a", names: []string{"b", "c"}, want: "a,b,c"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mergeTables(tc.list, tc.names)
			assert.Equal(t, tc.want, got, "mergeTables(%q, %v)", tc.list, tc.names)
		})
	}
}

// toSet returns the distinct values in all the lists.
func toSet(lists ...[]string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, list := range lists {
		for _, v := range list {
			if _, ok := seen[v]; !ok {
				seen[v] = struct{}{}
				out = append(out, v)
			}
		}
	}
	return out
}
