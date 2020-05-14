// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package filters

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

func TestEvaluation(t *testing.T) {
	type testCase struct {
		name      string
		condition string
		result    bool
	}

	testCases := []testCase{
		testCase{"simple version", "validate_version(%{[agent.version]}, '" + release.Version() + "')", true},
		testCase{"~ version release", "validate_version(%{[agent.version]}, '~" + release.Version() + "')", true},
		testCase{"^ version release", "validate_version(%{[agent.version]}, '^" + release.Version() + "')", true},
		testCase{"range to release", "validate_version(%{[agent.version]}, '1.0.0 - " + release.Version() + "')", true},
		testCase{"range lower", "validate_version(%{[agent.version]}, '1.0.0 - 5.0.0')", false},
		testCase{"range include", "validate_version(%{[agent.version]}, '1.0.0 - 100.0.0')", true},
		testCase{"family should equal", "%{[os.family]} == '" + runtime.GOOS + "'", true},
		testCase{"family should not equal", "%{[os.family]} != '" + runtime.GOOS + "'", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := evaluateConstraint(tc.condition)
			assert.NoError(t, err)
			assert.Equal(t, tc.result, r)

			// cleanup
			os.Remove("fleet.yml")
			os.Remove("fleet.yml.old")
		})
	}
}
