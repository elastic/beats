// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build race

package awss3

const (
	raceBuildEnabled = true
	// Race instrumentation slows decoding by roughly an order of magnitude, so
	// deadlines meant as safety nets have to grow with it.
	raceTimeoutScale = 10
)
