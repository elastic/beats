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

package mage

import (
	"fmt"
	"regexp"
	"strconv"
)

var SemanticVersionRegex = regexp.MustCompile(`(?m)^(\d+)\.(\d+)(?:\.(\d+))?`)

type SemanticVersion struct {
	Major, Minor, Patch int
}

// NewSemanticVersion return a new SemanticVersion parsed from string in the
// format of 'x.y' or 'x.y.z'.
func NewSemanticVersion(s string) (*SemanticVersion, error) {
	matches := SemanticVersionRegex.FindStringSubmatch(s)
	if len(matches) < 4 {
		return nil, fmt.Errorf("invalid version format %q", s)
	}

	major, _ := strconv.Atoi(matches[1])
	Minor, _ := strconv.Atoi(matches[2])
	Patch, _ := strconv.Atoi(matches[3])
	return &SemanticVersion{major, Minor, Patch}, nil
}

// LessThan return true iff s is less than x.
func (s *SemanticVersion) LessThan(x *SemanticVersion) bool {
	if s.Major != x.Major {
		return s.Major < x.Major
	}
	if s.Minor != x.Minor {
		return s.Minor < x.Minor
	}
	return s.Patch < x.Patch
}

// LessThanOrEqual return true iff s is less than or equal to x.
func (s *SemanticVersion) LessThanOrEqual(x *SemanticVersion) bool {
	if s.LessThan(x) {
		return true
	}
	return !x.LessThan(s)
}

func (s SemanticVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", s.Major, s.Minor, s.Patch)
}
