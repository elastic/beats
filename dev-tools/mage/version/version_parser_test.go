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

package version

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSemVerRegexpCompiles(t *testing.T) {
	require.NotNil(t, semVerFmtRegEx)
	assert.Contains(t, namedGroups, "major")
	assert.Contains(t, namedGroups, "minor")
	assert.Contains(t, namedGroups, "patch")
	assert.Contains(t, namedGroups, "prerelease")
	assert.Contains(t, namedGroups, "buildmetadata")
}

func TestParseVersion(t *testing.T) {
	type expected struct {
		parsed            *ParsedSemVer
		versionPrerelease string
		err               error
	}

	testcases := []struct {
		name     string
		input    string
		expected expected
	}{
		{
			name:  "Simple semver",
			input: "1.2.3",
			expected: expected{
				parsed: &ParsedSemVer{
					original:      "1.2.3",
					major:         1,
					minor:         2,
					patch:         3,
					prerelease:    "",
					buildMetadata: "",
				},
				versionPrerelease: "1.2.3",
				err:               nil,
			},
		},
		{
			name:  "Biiig semver",
			input: "1111.2222.3333",
			expected: expected{
				parsed: &ParsedSemVer{
					original:      "1111.2222.3333",
					major:         1111,
					minor:         2222,
					patch:         3333,
					prerelease:    "",
					buildMetadata: "",
				},
				versionPrerelease: "1111.2222.3333",
				err:               nil,
			},
		},
		{
			name:  "Simple semver with spaces around",
			input: " \t1.2.3 \r\n ",
			expected: expected{
				parsed: &ParsedSemVer{
					original:      " \t1.2.3 \r\n ",
					major:         1,
					minor:         2,
					patch:         3,
					prerelease:    "",
					buildMetadata: "",
				},
				versionPrerelease: "1.2.3",
				err:               nil,
			},
		},
		{
			name:  "Semver with prerelease",
			input: "1.2.3-SNAPSHOT",
			expected: expected{
				parsed: &ParsedSemVer{
					original:      "1.2.3-SNAPSHOT",
					major:         1,
					minor:         2,
					patch:         3,
					prerelease:    "SNAPSHOT",
					buildMetadata: "",
				},
				versionPrerelease: "1.2.3-SNAPSHOT",
				err:               nil,
			},
		},
		{
			name:  "Semver with versioned prerelease",
			input: "1.2.3-er.1+abcdef",
			expected: expected{
				parsed: &ParsedSemVer{
					original:      "1.2.3-er.1+abcdef",
					major:         1,
					minor:         2,
					patch:         3,
					prerelease:    "er.1",
					buildMetadata: "abcdef",
				},
				versionPrerelease: "1.2.3-er.1",
				err:               nil,
			},
		},
		{
			name:  "Semver with prerelease and build metadata",
			input: "1.2.3-SNAPSHOT+abcdef",
			expected: expected{
				parsed: &ParsedSemVer{
					original:      "1.2.3-SNAPSHOT+abcdef",
					major:         1,
					minor:         2,
					patch:         3,
					prerelease:    "SNAPSHOT",
					buildMetadata: "abcdef",
				},
				versionPrerelease: "1.2.3-SNAPSHOT",
				err:               nil,
			},
		},
		{
			name:  "Semver string version, with double prerelease(er and snapshot)",
			input: "1.2.5-er.1-SNAPSHOT",
			expected: expected{
				parsed: &ParsedSemVer{
					original:      "1.2.5-er.1-SNAPSHOT",
					major:         1,
					minor:         2,
					patch:         5,
					prerelease:    "er.1-SNAPSHOT",
					buildMetadata: "",
				},
				versionPrerelease: "1.2.5-er.1-SNAPSHOT",
			},
		},
		{
			name:  "Error truncated semver",
			input: "2.3",
			expected: expected{
				parsed: nil,
				err:    ErrNoMatch,
			},
		},
		{
			name:  "Error missing prerelease type",
			input: "1.2.3-",
			expected: expected{
				parsed: nil,
				err:    ErrNoMatch,
			},
		},
		{
			name:  "Error missing build metadata",
			input: "1.2.3-beta.22+",
			expected: expected{
				parsed: nil,
				err:    ErrNoMatch,
			},
		},
		{
			name:  "Weird random string version",
			input: "asdasdasdasdasd",
			expected: expected{
				parsed: nil,
				err:    ErrNoMatch,
			},
		},
		{
			name:  "Almost semver string version, with double extra meta separator",
			input: "1.2.3++",
			expected: expected{
				parsed: nil,
				err:    ErrNoMatch,
			},
		},
		{
			name:  "Almost semver string version, with empty minor version",
			input: "1..2+ab",
			expected: expected{
				parsed: nil,
				err:    ErrNoMatch,
			},
		},
		{
			name:  "Almost semver string version, with patch containing non-digits",
			input: "1.2.5ab0",
			expected: expected{
				parsed: nil,
				err:    ErrNoMatch,
			},
		},
		{
			name:  "Split string version",
			input: "4.5\r\n.6",
			expected: expected{
				parsed: nil,
				err:    ErrNoMatch,
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actualParsed, err := ParseVersion(tc.input)

			if tc.expected.err != nil {
				assert.ErrorIs(t, err, tc.expected.err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected.parsed, actualParsed)

			// assert getters
			assert.Equal(t, tc.expected.parsed.original, actualParsed.Original())
			assert.Equal(t, tc.expected.parsed.major, actualParsed.Major())
			assert.Equal(t, tc.expected.parsed.minor, actualParsed.Minor())
			assert.Equal(t, tc.expected.parsed.patch, actualParsed.Patch())
			assert.Equal(t, tc.expected.parsed.prerelease, actualParsed.Prerelease())
			assert.Equal(t, tc.expected.parsed.buildMetadata, actualParsed.BuildMetadata())
			assert.Equal(t, tc.expected.versionPrerelease, actualParsed.VersionWithPrerelease())

			// verify that String() method returns the same input string (after trimming)
			assert.Equal(t, strings.TrimSpace(tc.input), actualParsed.String())
		})
	}
}

func TestIsSnapshot(t *testing.T) {
	testcases := []struct {
		name     string
		input    string
		snapshot bool
	}{
		{
			name:     "Simple snapshot",
			input:    "8.8.0-SNAPSHOT",
			snapshot: true,
		},
		{
			name:     "Snapshot with build meta",
			input:    "8.8.0-SNAPSHOT+abcdef",
			snapshot: true,
		},
		{
			name:     "Snapshot comparison is case sensitive",
			input:    "8.8.0-sNapShOt",
			snapshot: false,
		},
		{
			name:     "Only major minor patch",
			input:    "8.8.0",
			snapshot: false,
		},
		{
			name:     "Alpha prerelease is not snapshot",
			input:    "8.8.0-alpha",
			snapshot: false,
		},
		{
			name:     "Emergency release is not snapshot",
			input:    "8.8.0-er.1",
			snapshot: false,
		},
		{
			name:     "Emergency release snapshot is actually a snapshot",
			input:    "8.8.0-er.1-SNAPSHOT ",
			snapshot: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			psv, err := ParseVersion(tc.input)
			require.NoError(t, err)
			require.NotNil(t, psv)
			assert.Equal(t, tc.snapshot, psv.IsSnapshot())
		})

	}

}

func TestLess(t *testing.T) {
	testcases := []struct {
		name         string
		leftVersion  string
		rightVersion string
		less         bool
	}{
		{
			name:         "major version less than ours",
			leftVersion:  "7.17.10",
			rightVersion: "8.9.0",
			less:         true,
		},
		{
			name:         "minor version less than ours",
			leftVersion:  "8.6.2",
			rightVersion: "8.9.0",
			less:         true,
		},
		{
			name:         "patch version less than ours",
			leftVersion:  "8.7.0",
			rightVersion: "8.7.1",
			less:         true,
		},
		{
			name:         "prerelease is always less than non-prerelease",
			leftVersion:  "8.9.0-SNAPSHOT",
			rightVersion: "8.9.0",
			less:         true,
		},
		{
			name:         "2 prereleases have no specific order",
			leftVersion:  "8.9.0-SNAPSHOT",
			rightVersion: "8.9.0-er1",
			less:         false,
		},
		{
			name:         "2 prereleases have no specific order, reversed",
			leftVersion:  "8.9.0-er1",
			rightVersion: "8.9.0-SNAPSHOT",
			less:         false,
		},
		{
			name:         "build metadata have no influence on precedence",
			leftVersion:  "8.9.0-SNAPSHOT+aaaaaa",
			rightVersion: "8.9.0-SNAPSHOT+bbbbbb",
			less:         false,
		},
		{
			name:         "build metadata have no influence on precedence, reversed",
			leftVersion:  "8.9.0-SNAPSHOT+bbbbbb",
			rightVersion: "8.9.0-SNAPSHOT+aaaaaa",
			less:         false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			left, err := ParseVersion(tc.leftVersion)
			require.NoError(t, err)
			require.NotNil(t, left)
			right, err := ParseVersion(tc.rightVersion)
			require.NoError(t, err)
			require.NotNil(t, right)
			assert.Equal(t, left.Less(*right), tc.less)
		})
	}
}

func TestPreviousMinor(t *testing.T) {
	testcases := []struct {
		name             string
		version          string
		prevMinorVersion string
	}{
		{
			name:             "basic release version",
			version:          "8.7.0",
			prevMinorVersion: "8.6.0",
		},
		{
			name:             "snapshot release version",
			version:          "8.9.3-SNAPSHOT",
			prevMinorVersion: "8.8.0-SNAPSHOT",
		},
		{
			name:             "emergency release version",
			version:          "8.9.0-er1",
			prevMinorVersion: "8.8.0-er1",
		},
		{
			name:             "previous major version",
			version:          "8.0.0",
			prevMinorVersion: "7.17.10",
		},
		{
			name:             "previous major snapshot",
			version:          "8.0.0-SNAPSHOT",
			prevMinorVersion: "7.17.10-SNAPSHOT",
		},
		{
			name:             "snapshot version with metadata",
			version:          "8.9.1-SNAPSHOT+aaaaaa",
			prevMinorVersion: "8.8.0-SNAPSHOT+aaaaaa",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := ParseVersion(tc.version)
			require.NoError(t, err)
			require.NotNil(t, parsed)

			prev, err := parsed.GetPreviousMinor()
			require.NoError(t, err)
			require.Equal(t, tc.prevMinorVersion, prev.String())
		})
	}
}
