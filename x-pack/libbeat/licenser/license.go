// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package licenser

import (
	"time"
)

// License represents the license of this beat, the license is fetched and returned from
// the elasticsearch cluster.
//
// The x-pack endpoint returns the following JSON response.
//
// "license": {
//   "uid": "936183d8-f48c-4a3f-959a-a52aa2563279",
//   "type": "platinum",
//   "mode": "platinum",
//   "status": "active"
// },
//
// Definition:
// type is the installed license.
// mode is the license in operation. (effective license)
// status is the type installed is active or not.
type License struct {
	UUID        string      `json:"uid"`
	Type        LicenseType `json:"type"`
	Mode        LicenseType `json:"mode"`
	Status      State       `json:"status"`
	Features    features    `json:"features"`
	TrialExpiry expiryTime  `json:"expiry_date_in_millis,omitempty"`
}

// Features defines the list of features exposed by the elasticsearch cluster.
type features struct {
	Graph      graph      `json:"graph"`
	Logstash   logstash   `json:"logstash"`
	ML         ml         `json:"ml"`
	Monitoring monitoring `json:"monitoring"`
	Rollup     rollup     `json:"rollup"`
	Security   security   `json:"security"`
	Watcher    watcher    `json:"watcher"`
}

type expiryTime time.Time

// Base define the field common for every feature.
type Base struct {
	Enabled   bool `json:"enabled"`
	Available bool `json:"available"`
}

// Defines all the available features
type graph struct{ *Base }
type logstash struct{ *Base }
type ml struct{ *Base }
type monitoring struct{ *Base }
type rollup struct{ *Base }
type security struct{ *Base }
type watcher struct{ *Base }

// Get return the current license
func (l *License) Get() LicenseType {
	return l.Mode
}

// Cover returns true if the provided license is included in the range of license.
//
// Basic -> match basic, gold and platinum
// gold -> match gold and platinum
// platinum -> match  platinum only
func (l *License) Cover(license LicenseType) bool {
	if l.Mode >= license {
		return true
	}
	return false
}

// Is returns true if the provided license is an exact match.
func (l *License) Is(license LicenseType) bool {
	return l.Mode == license
}

// IsActive returns true if the current license from the server is active.
func (l *License) IsActive() bool {
	return l.Status == Active
}

// IsTrial returns true if the remote cluster is in trial mode.
func (l *License) IsTrial() bool {
	return l.Mode == Trial
}

// IsTrialExpired returns false if the we are not in trial mode and when we are in trial mode
// we check for the expiry data.
func (l *License) IsTrialExpired() bool {
	if !l.IsTrial() {
		return false
	}

	if time.Time(l.TrialExpiry).Sub(time.Now()) > 0 {
		return false
	}

	return true
}

// EqualTo returns true if the two license are the same, we compare license to reduce the number
// message send to the watchers.
func (l *License) EqualTo(other *License) bool {
	return l.UUID == other.UUID &&
		l.Type == other.Type &&
		l.Mode == other.Mode &&
		l.Status == other.Status
}
