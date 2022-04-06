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

// These fields contain Windows Portable Executable (PE) metadata.
type Pe struct {
	// Internal name of the file, provided at compile-time.
	OriginalFileName string `ecs:"original_file_name"`

	// Internal version of the file, provided at compile-time.
	FileVersion string `ecs:"file_version"`

	// Internal description of the file, provided at compile-time.
	Description string `ecs:"description"`

	// Internal product name of the file, provided at compile-time.
	Product string `ecs:"product"`

	// Internal company name of the file, provided at compile-time.
	Company string `ecs:"company"`

	// A hash of the imports in a PE file. An imphash -- or import hash -- can
	// be used to fingerprint binaries even after recompilation or other
	// code-level transformations have occurred, which would change more
	// traditional hash values.
	// Learn more at
	// https://www.fireeye.com/blog/threat-research/2014/01/tracking-malware-import-hashing.html.
	Imphash string `ecs:"imphash"`

	// CPU architecture target for the file.
	Architecture string `ecs:"architecture"`
}
