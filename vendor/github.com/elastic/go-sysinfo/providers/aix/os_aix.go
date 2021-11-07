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

package aix

import (
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/go-sysinfo/types"
)

// OperatingSystem returns information of the host operating system
func OperatingSystem() (*types.OSInfo, error) {
	return getOSInfo()
}

func getOSInfo() (*types.OSInfo, error) {
	major, minor, err := getKernelVersion()
	if err != nil {
		return nil, err
	}

	// Retrieve build version from "/proc/version".
	procVersion, err := ioutil.ReadFile("/proc/version")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get OS info: cannot open /proc/version")
	}
	build := strings.SplitN(string(procVersion), "\n", 4)[2]

	return &types.OSInfo{
		Type:     "unix",
		Family:   "aix",
		Platform: "aix",
		Name:     "aix",
		Version:  strconv.Itoa(major) + "." + strconv.Itoa(minor),
		Major:    major,
		Minor:    minor,
		Patch:    0, // No patch version
		Build:    build,
	}, nil
}
