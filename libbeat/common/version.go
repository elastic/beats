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

package common

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	version string
	Major   int
	Minor   int
	Bugfix  int
	Meta    string
}

// MustNewVersion creates a version from the given version string.
// If the version string is invalid, MustNewVersion panics.
func MustNewVersion(version string) *Version {
	v, err := NewVersion(version)
	if err != nil {
		panic(err)
	}
	return v
}

// NewVersion expects a string in the format:
// major.minor.bugfix(-meta)
func NewVersion(version string) (*Version, error) {
	v := Version{
		version: version,
	}

	// Check for meta info
	if strings.Contains(version, "-") {
		tmp := strings.Split(version, "-")
		version = tmp[0]
		v.Meta = tmp[1]
	}

	versions := strings.Split(version, ".")
	if len(versions) != 3 {
		return nil, fmt.Errorf("Passed version is not semver: %s", version)
	}

	var err error
	v.Major, err = strconv.Atoi(versions[0])
	if err != nil {
		return nil, fmt.Errorf("Could not convert major to integer: %s", versions[0])
	}

	v.Minor, err = strconv.Atoi(versions[1])
	if err != nil {
		return nil, fmt.Errorf("Could not convert minor to integer: %s", versions[1])
	}

	v.Bugfix, err = strconv.Atoi(versions[2])
	if err != nil {
		return nil, fmt.Errorf("Could not convert bugfix to integer: %s", versions[2])
	}

	return &v, nil
}

// IsValid returns true if the version object stores a successfully parsed version number.
func (v *Version) IsValid() bool {
	return v.version != ""
}

func (v *Version) IsMajor(major int) bool {
	return major == v.Major
}

// LessThan returns true if v is strictly smaller than v1. When comparing, the major,
// minor, bugfix and pre-release numbers are compared in order.
func (v *Version) LessThanOrEqual(withMeta bool, v1 *Version) bool {
	if withMeta && v.version == v1.version {
		return true
	}
	if v.Major < v1.Major {
		return true
	}
	if v.Major == v1.Major {
		if v.Minor < v1.Minor {
			return true
		}
		if v.Minor == v1.Minor {
			if v.Bugfix < v1.Bugfix {
				return true
			}
			if v.Bugfix == v1.Bugfix {
				if withMeta {
					return v.metaIsLessThanOrEqual(v1)
				} else {
					return true
				}
			}
		}
	}
	return false
}

// LessThan returns true if v is strictly smaller than v1. When comparing, the major,
// minor and bugfix numbers are compared in order. The meta part is not taken into account.
func (v *Version) LessThan(v1 *Version) bool {
	lessThan := v.LessThanMajorMinor(v1)
	if lessThan {
		return true
	}

	if v.Minor == v1.Minor {
		if v.Bugfix < v1.Bugfix {
			return true
		}
	}

	return false
}

// LessThanMajorMinor returns true if v is smaller or equal to v1 based on the major and minor version. The bugfix version and meta part are not taken into account.
// minor numbers are compared in order. The and bugfix version and meta part is not taken into account.
func (v *Version) LessThanMajorMinor(v1 *Version) bool {
	if v.Major < v1.Major {
		return true
	} else if v.Major == v1.Major {
		if v.Minor < v1.Minor {
			return true
		}
	}
	return false
}

func (v *Version) String() string {
	return v.version
}

func (v *Version) metaIsLessThanOrEqual(v1 *Version) bool {
	if v.Meta == "" && v1.Meta == "" {
		return true
	}
	if v.Meta == "" {
		return false
	}
	if v1.Meta == "" {
		return true
	}
	return v.Meta <= v1.Meta
}

// UnmarshalJSON unmarshals a JSON string version representation into a Version struct
// Implements https://golang.org/pkg/encoding/json/#Unmarshaler
func (v *Version) UnmarshalJSON(version []byte) error {
	var versionStr string
	err := json.Unmarshal(version, &versionStr)
	if err != nil {
		return err
	}

	ver, err := NewVersion(versionStr)
	if err != nil {
		return err
	}

	if ver == nil {
		return fmt.Errorf("could not unmarshal version from JSON")
	}

	*v = *ver
	return nil
}
