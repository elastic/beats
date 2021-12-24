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

package ecs

// These fields contain information about code libraries dynamically loaded
// into processes.
//
// Many operating systems refer to "shared code libraries" with different
// names, but this field set refers to all of the following:
// * Dynamic-link library (`.dll`) commonly used on Windows
// * Shared Object (`.so`) commonly used on Unix-like operating systems
// * Dynamic library (`.dylib`) commonly used on macOS
type Dll struct {
	// Name of the library.
	// This generally maps to the name of the file on disk.
	Name string `ecs:"name"`

	// Full file path of the library.
	Path string `ecs:"path"`
}
