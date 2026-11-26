// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package osqd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
)

// TestArgsUserTables checks how the user's disable_tables and enable_tables
// flags combine with the default disabled tables. The cases hold in builds with
// and without the requirefips tag, because FIPS builds only add tables to the
// disabled list.
func TestArgsUserTables(t *testing.T) {
	tests := []struct {
		name  string
		flags Flags
		// disabled are tables that must be disabled, and enabled are tables
		// that must not be.
		disabled []string
		enabled  []string
	}{
		{
			name:     "defaults_only",
			disabled: []string{"carves", "curl"},
		},
		{
			name:     "user_disables_a_table",
			flags:    Flags{flagDisableTables: "processes"},
			disabled: []string{"carves", "curl", "processes"},
		},
		{
			name:     "user_disables_a_table_and_enables_another",
			flags:    Flags{flagDisableTables: "processes", flagEnableTables: "users"},
			disabled: []string{"carves", "curl", "processes"},
			enabled:  []string{"users"},
		},
		{
			// The enable flag removes a table from the disabled list, so
			// disabling and enabling the same table leaves it enabled. Here
			// nothing else is disabled, so the merged list is empty.
			name:    "user_disables_and_enables_the_same_table_and_the_defaults",
			flags:   Flags{flagDisableTables: "processes", flagEnableTables: "carves,curl,processes"},
			enabled: []string{"processes"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			osq, err := New("/var/run/foobar", WithLogger(logptest.NewTestingLogger(t, "osqueryd_test")))
			require.NoError(t, err, "creating osqueryd")

			args := osq.args(tc.flags)

			got := disabledTablesFromArgs(args)
			for _, name := range tc.disabled {
				assert.Contains(t, got, name, "table %s must be disabled in %v", name, args)
			}
			for _, name := range tc.enabled {
				assert.NotContains(t, got, name, "table %s must not be disabled in %v", name, args)
			}
		})
	}
}

// disabledTablesFromArgs returns the tables named by the disable_tables flag.
func disabledTablesFromArgs(args []string) []string {
	const prefix = "--" + flagDisableTables + "="
	for _, a := range args {
		if list, ok := strings.CutPrefix(a, prefix); ok {
			return strings.Split(list, ",")
		}
	}
	return nil
}
