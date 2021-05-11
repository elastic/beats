// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package management

const (
	// ModeCentralManagement is a default CM mode, using existing processes.
	ModeCentralManagement = "x-pack-cm" // TODO remove?

	// ModeFleet is a management mode where fleet is used to retrieve configurations.
	ModeFleet = "x-pack-fleet"
)
