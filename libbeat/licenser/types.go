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

package licenser

// LicenseType defines what kind of license is currently available.
type LicenseType int

//go:generate stringer -type=LicenseType -linecomment=true
const (
	OSS      LicenseType = iota // Open source
	Trial                       // Trial
	Basic                       // Basic
	Gold                        // Gold
	Platinum                    // Platinum
)

// State of the license can be active or inactive.
type State int

//go:generate stringer -type=State
const (
	Inactive State = iota
	Active
)
