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

package testutils

import (
	"runtime/debug"
	"strings"
)

// FIPSBuildInfo captures the FIPS-related settings found in a binary's build info.
type FIPSBuildInfo struct {
	TagsFound               bool
	TagsHaveRequireFIPS     bool
	GOFIPS140Found          bool
	GOFIPS140Value          string
	GOFIPS140IsCertified    bool // value starts with the certified module version, e.g. "v1.0.0"
	DefaultGODEBUGFound     bool
	DefaultGODEBUGHasFIPSOn bool
}

// CheckFIPSBuildInfo scans a binary's build settings (as reported by debug/buildinfo)
// for the markers that indicate a FIPS-compliant build: the "requirefips" build tag,
// a GOFIPS140 setting referencing the certified module version, and a DefaultGODEBUG
// setting that enables fips140=on at runtime.
func CheckFIPSBuildInfo(settings []debug.BuildSetting) FIPSBuildInfo {
	var info FIPSBuildInfo
	for _, setting := range settings {
		switch setting.Key {
		case "-tags":
			info.TagsFound = true
			info.TagsHaveRequireFIPS = strings.Contains(setting.Value, "requirefips")
		case "GOFIPS140":
			info.GOFIPS140Found = true
			info.GOFIPS140Value = setting.Value
			info.GOFIPS140IsCertified = strings.HasPrefix(setting.Value, "v1.0.0")
		case "DefaultGODEBUG":
			info.DefaultGODEBUGFound = true
			for entry := range strings.SplitSeq(setting.Value, ",") {
				if key, val, ok := strings.Cut(entry, "="); ok && key == "fips140" && val == "on" {
					info.DefaultGODEBUGHasFIPSOn = true
					break
				}
			}
		}
	}
	return info
}
