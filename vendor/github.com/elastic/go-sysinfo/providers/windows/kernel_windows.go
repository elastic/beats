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

package windows

import (
	"github.com/elastic/go-windows"
)

const windowsKernelExe = `C:\Windows\System32\ntoskrnl.exe`

func KernelVersion() (string, error) {
	versionData, err := windows.GetFileVersionInfo(windowsKernelExe)
	if err != nil {
		return "", err
	}

	fileVersion, err := versionData.QueryValue("FileVersion")
	if err == nil {
		return fileVersion, nil
	}

	// Make a second attempt through the fixed version info.
	info, err := versionData.FixedFileInfo()
	if err != nil {
		return "", err
	}
	return info.ProductVersion(), nil
}
