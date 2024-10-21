// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License 2.0;
// you may not use this file except in compliance with the Elastic License 2.0.

//go:build integration

package integration

import "github.com/elastic/elastic-agent/pkg/testing/define"

const (
	// Default group.
	Default = define.Default

	// Fleet group of tests. Used for testing Elastic Agent with Fleet.
	Fleet = "fleet"

	// FleetPrivileged group of tests. Used for testing Elastic Agent with Fleet installed privileged.
	FleetPrivileged = "fleet-privileged"

	// FleetAirgapped group of tests. Used for testing Elastic Agent with Fleet and airgapped.
	FleetAirgapped = "fleet-airgapped"

	// FleetAirgappedPrivileged group of tests. Used for testing Elastic Agent with Fleet installed
	// privileged and airgapped.
	FleetAirgappedPrivileged = "fleet-airgapped-privileged"

	// FleetUpgradeToPRBuild group of tests. Used for testing Elastic Agent
	// upgrading to a build built from the PR being tested.
	FleetUpgradeToPRBuild = "fleet-upgrade-to-pr-build"

	// FQDN group of tests. Used for testing Elastic Agent with FQDN enabled.
	FQDN = "fqdn"

	// Upgrade group of tests. Used for testing upgrades.
	Upgrade = "upgrade"

	// Deb group of tests. Used for testing .deb packages install & upgrades
	Deb = "deb"

	// RPM group of tests. Used for testing .rpm packages install & upgrades
	RPM = "rpm"
)
