// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package osqd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/pkg/logp/logptest"
)

// TestArgsNoFIPSLeavesFlagsAlone checks that builds without the requirefips
// tag do not pass the flags that FIPS builds enforce.
func TestArgsNoFIPSLeavesFlagsAlone(t *testing.T) {
	osq, err := newOsqueryD("/var/run/foobar", WithLogger(logptest.NewTestingLogger(t, "osqueryd_test")))
	require.NoError(t, err, "creating osqueryd")

	args := osq.args(Flags{"disable_distributed": false})

	assert.Contains(t, args, "--disable_distributed=false", "user flags must pass through in builds without requirefips")

	for _, name := range []string{"disable_enrollment", "disable_carver", "aws_enforce_fips", "yara_sigurl_authenticate"} {
		for _, a := range args {
			assert.False(t, strings.HasPrefix(a, "--"+name+"="), "unexpected FIPS flag %s", a)
		}
	}

	_, defaults := getEnabledDisabledTables(nil)
	assert.Contains(t, args, "--disable_tables="+strings.Join(defaults, ","), "default disabled tables must not change")
}

// TestEnforceFIPSOptionsNoFIPSLeavesOptionsAlone checks that builds without the
// requirefips tag return the options unchanged.
func TestEnforceFIPSOptionsNoFIPSLeavesOptionsAlone(t *testing.T) {
	options := map[string]any{"disable_distributed": false, "schedule_splay_percent": 5}

	got := EnforceFIPSOptions(options)

	assert.Equal(t, map[string]any{"disable_distributed": false, "schedule_splay_percent": 5}, got, "options must pass through in builds without requirefips")
	assert.Nil(t, EnforceFIPSOptions(nil), "nil options must stay nil in builds without requirefips")
}
