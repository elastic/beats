// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package licenser

import (
	"github.com/elastic/beats/v7/libbeat/logp"
)

// CheckFunc signature to implement a function that validate a license.
type CheckFunc func(*logp.Logger, License) bool

// CheckTrial returns true if the license is in trial and the license is not expired.
func CheckTrial(log *logp.Logger, license License) bool {
	log.Debug("Checking trial license")
	if license.IsTrial() {
		if license.IsTrialExpired() {
			log.Error("Trial license is expired")
			return false
		}
		log.Info("Trial license active")
		return true
	}
	return false
}

// CheckLicenseCover check that the current license cover the requested license.
func CheckLicenseCover(licenseType LicenseType) func(*logp.Logger, License) bool {
	return func(log *logp.Logger, license License) bool {
		log.Debug("Checking that license covers %s", licenseType)
		if license.Cover(licenseType) && license.IsActive() {
			return true
		}
		log.Infof("License is active for %s", licenseType)
		return false
	}
}

// CheckBasic returns true if the license is
var CheckBasic = CheckLicenseCover(Basic)

// Validate uses a set of checks to validate if a license is valid or not and will return true on on the
// first check that validate the license.
func Validate(log *logp.Logger, license License, checks ...CheckFunc) bool {
	for _, check := range checks {
		if check(log, license) {
			return true
		}
	}
	return false
}

// BasicAndAboveOrTrial return true if the license is basic or if the license is trial and active.
func BasicAndAboveOrTrial(log *logp.Logger, license License) bool {
	return CheckBasic(log, license) || CheckTrial(log, license)
}
