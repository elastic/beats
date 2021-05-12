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

import "fmt"

// LicenseType defines what kind of license is currently available.
type LicenseType int

//go:generate stringer -type=LicenseType -linecomment=true
const (
	OSS        LicenseType = iota // Open source
	Trial                         // Trial
	Basic                         // Basic
	Standard                      // Standard
	Gold                          // Gold
	Platinum                      // Platinum
	Enterprise                    // Enterprise
)

// State of the license can be active or inactive.
type State int

//go:generate stringer -type=State
const (
	Inactive State = iota
	Active
	Expired
)

var stateLookup = map[string]State{
	"inactive": Inactive,
	"active":   Active,
	"expired":  Expired,
}

var licenseLookup = map[string]LicenseType{
	"oss":        OSS,
	"trial":      Trial,
	"standard":   Standard,
	"basic":      Basic,
	"gold":       Gold,
	"platinum":   Platinum,
	"enterprise": Enterprise,
}

// UnmarshalJSON takes a bytes array and convert it to the appropriate license type.
func (t *LicenseType) UnmarshalJSON(b []byte) error {
	if len(b) <= 2 {
		return fmt.Errorf("invalid string for license type, received: '%s'", string(b))
	}
	s := string(b[1 : len(b)-1])
	if license, ok := licenseLookup[s]; ok {
		*t = license
		return nil
	}

	return fmt.Errorf("unknown license type, received: '%s'", s)
}

// UnmarshalJSON takes a bytes array and convert it to the appropriate state.
func (st *State) UnmarshalJSON(b []byte) error {
	// we are only interested in the content between the quotes.
	if len(b) <= 2 {
		return fmt.Errorf("invalid string for state, received: '%s'", string(b))
	}

	s := string(b[1 : len(b)-1])
	if state, ok := stateLookup[s]; ok {
		*st = state
		return nil
	}
	return fmt.Errorf("unknown state, received: '%s'", s)
}
