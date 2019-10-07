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

package windows

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/sys/windows/registry"

	"github.com/elastic/go-sysinfo/types"
)

func OperatingSystem() (*types.OSInfo, error) {
	const key = registry.LOCAL_MACHINE
	const path = `SOFTWARE\Microsoft\Windows NT\CurrentVersion`
	const flags = registry.READ | registry.WOW64_64KEY

	k, err := registry.OpenKey(key, path, flags)
	if err != nil {
		return nil, errors.Wrapf(err, `failed to open HKLM\%v`, path)
	}
	defer k.Close()

	osInfo := &types.OSInfo{
		Family:   "windows",
		Platform: "windows",
	}
	name := "ProductName"
	osInfo.Name, _, err = k.GetStringValue(name)
	if err != nil {
		return nil, errors.Wrapf(err, `failed to get value of HKLM\%v\%v`, path, name)
	}

	// Newer versions (Win 10 and 2016) have CurrentMajor/CurrentMinor.
	major, _, majorErr := k.GetIntegerValue("CurrentMajorVersionNumber")
	minor, _, minorErr := k.GetIntegerValue("CurrentMinorVersionNumber")
	if majorErr == nil && minorErr == nil {
		osInfo.Major = int(major)
		osInfo.Minor = int(minor)
		osInfo.Version = fmt.Sprintf("%d.%d", major, minor)
	} else {
		name = "CurrentVersion"
		osInfo.Version, _, err = k.GetStringValue(name)
		if err != nil {
			return nil, errors.Wrapf(err, `failed to get value of HKLM\%v\%v`, path, name)
		}
		parts := strings.SplitN(osInfo.Version, ".", 3)
		for i, p := range parts {
			switch i {
			case 0:
				osInfo.Major, _ = strconv.Atoi(p)
			case 1:
				osInfo.Major, _ = strconv.Atoi(p)
			}
		}
	}

	name = "CurrentBuild"
	osInfo.Build, _, err = k.GetStringValue(name)
	if err != nil {
		return nil, errors.Wrapf(err, `failed to get value of HKLM\%v\%v`, path, name)
	}

	// Update Build Revision (optional)
	name = "UBR"
	updateBuildRevision, _, err := k.GetIntegerValue(name)
	if err != nil && err != registry.ErrNotExist {
		return nil, errors.Wrapf(err, `failed to get value of HKLM\%v\%v`, path, name)
	} else {
		osInfo.Build = fmt.Sprintf("%v.%d", osInfo.Build, updateBuildRevision)
	}

	return osInfo, nil
}
