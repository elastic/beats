// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package licenser

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
