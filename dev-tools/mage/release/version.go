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

package release

import (
	"fmt"

	"github.com/Masterminds/semver"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
)

func parseReleaseVersion(raw string) (*semver.Version, error) {
	return devtools.ParseReleaseVersion(raw)
}

// selectLatestReleaseBefore picks the highest same-major version strictly less than current.
func selectLatestReleaseBefore(versions []string, currentVersion string) (string, error) {
	current, err := parseReleaseVersion(currentVersion)
	if err != nil {
		return "", err
	}

	var best *semver.Version
	for _, raw := range versions {
		candidate, err := parseReleaseVersion(raw)
		if err != nil {
			continue
		}
		if candidate.Major() != current.Major() {
			continue
		}
		if !candidate.LessThan(current) {
			continue
		}
		if best == nil || best.LessThan(candidate) {
			best = candidate
		}
	}
	if best == nil {
		return "", fmt.Errorf("no published release found before %s (same major)", currentVersion)
	}
	return best.String(), nil
}

func inferReleaseBranch(currentRelease string) string {
	v, err := parseReleaseVersion(currentRelease)
	if err != nil {
		return ""
	}
	return devtools.InferReleaseBranch(v)
}

func inferLatestRelease(currentRelease string) (string, error) {
	v, err := parseReleaseVersion(currentRelease)
	if err != nil {
		return "", err
	}
	return devtools.InferLatestRelease(v), nil
}

func inferNextRelease(currentRelease string) string {
	v, err := parseReleaseVersion(currentRelease)
	if err != nil {
		return ""
	}
	return devtools.InferNextRelease(v)
}

func inferNextProjectMinorVersion(currentRelease string) (string, error) {
	v, err := parseReleaseVersion(currentRelease)
	if err != nil {
		return "", err
	}
	return devtools.InferNextProjectMinorVersion(v), nil
}

func inferNextProjectMinorBranch(currentRelease string) string {
	v, err := parseReleaseVersion(currentRelease)
	if err != nil {
		return ""
	}
	return devtools.InferNextProjectMinorBranch(v)
}
