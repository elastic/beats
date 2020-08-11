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
//{
// "license" : {
//   "status" : "active",
//   "uid" : "cbff45e7-c553-41f7-ae4f-9205eabd80xx",
//   "type" : "trial",
//   "issue_date" : "2018-10-20T22:05:12.332Z",
//   "issue_date_in_millis" : 1540073112332,
//   "expiry_date" : "2018-11-19T22:05:12.332Z",
//   "expiry_date_in_millis" : 1542665112332,
//   "max_nodes" : 1000,
//   "issued_to" : "test",
//   "issuer" : "elasticsearch",
//   "start_date_in_millis" : -1
// }
// }
// Definition:
// type is the installed license.
// mode is the license in operation. (effective license)
// status is the type installed is active or not.
type License struct {
	UUID        string      `json:"uid"`
	Type        LicenseType `json:"type"`
	Status      State       `json:"status"`
	TrialExpiry expiryTime  `json:"expiry_date_in_millis,omitempty"`
}

type expiryTime time.Time

// Get returns the license type.
func (l *License) Get() LicenseType {
	return l.Type
}

// Cover returns true if the provided license is included in the range of license.
//
// Basic -> match basic, gold and platinum
// gold -> match gold and platinum
// platinum -> match  platinum only
func (l *License) Cover(license LicenseType) bool {
	if l.Type >= license {
		return true
	}
	return false
}

// Is returns true if the provided license is an exact match.
func (l *License) Is(license LicenseType) bool {
	return l.Type == license
}

// IsActive returns true if the current license from the server is active.
func (l *License) IsActive() bool {
	return l.Status == Active
}

// IsTrial returns true if the remote cluster is in trial mode.
func (l *License) IsTrial() bool {
	return l.Type == Trial
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
		l.Status == other.Status
}
