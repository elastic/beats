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

package file

import (
	"os"
)

type ExtendedFileInfo interface {
	os.FileInfo
	GetOSState() StateOS
}

type extendedFileInfo struct {
	os.FileInfo
	osSpecific *StateOS
}

// GetOSState returns the platform specific StateOS.
// The data is fetched once and cached.
func (f *extendedFileInfo) GetOSState() StateOS {
	if f == nil || f.FileInfo == nil {
		return StateOS{}
	}

	if f.osSpecific != nil {
		return *f.osSpecific
	}

	osSpecific := GetOSState(f.FileInfo)
	f.osSpecific = &osSpecific
	return osSpecific
}

// ExtendFileInfo wraps the standard FileInfo with an extended version.
func ExtendFileInfo(fi os.FileInfo) ExtendedFileInfo {
	return &extendedFileInfo{FileInfo: fi}
}
