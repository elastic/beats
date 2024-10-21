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

// Build describes a build and its paths.
type Build struct {
	// Version of the Elastic Agent build.
	Version string
	// Type of OS this build is for.
	Type string
	// Arch is architecture this build is for.
	Arch string
	// Path is the path to the build.
	Path string
	// SHA512 is the path to the SHA512 file.
	SHA512Path string
}
