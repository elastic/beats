// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package darwin

import (
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"howett.net/plist"

	"github.com/elastic/go-sysinfo/types"
)

const (
	systemVersionPlist = "/System/Library/CoreServices/SystemVersion.plist"

	plistProductName         = "ProductName"
	plistProductVersion      = "ProductVersion"
	plistProductBuildVersion = "ProductBuildVersion"
)

func OperatingSystem() (*types.OSInfo, error) {
	data, err := ioutil.ReadFile(systemVersionPlist)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read plist file")
	}

	return getOSInfo(data)
}

func getOSInfo(data []byte) (*types.OSInfo, error) {
	attrs := map[string]string{}
	if _, err := plist.Unmarshal(data, &attrs); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal plist data")
	}

	productName, found := attrs[plistProductName]
	if !found {
		return nil, errors.Errorf("plist key %v not found", plistProductName)
	}

	version, found := attrs[plistProductVersion]
	if !found {
		return nil, errors.Errorf("plist key %v not found", plistProductVersion)
	}

	build, found := attrs[plistProductBuildVersion]
	if !found {
		return nil, errors.Errorf("plist key %v not found", plistProductBuildVersion)
	}

	var major, minor, patch int
	for i, v := range strings.SplitN(version, ".", 3) {
		switch i {
		case 0:
			major, _ = strconv.Atoi(v)
		case 1:
			minor, _ = strconv.Atoi(v)
		case 2:
			patch, _ = strconv.Atoi(v)
		default:
			break
		}
	}

	return &types.OSInfo{
		Family:   "darwin",
		Platform: "darwin",
		Name:     productName,
		Version:  version,
		Major:    major,
		Minor:    minor,
		Patch:    patch,
		Build:    build,
	}, nil
}
