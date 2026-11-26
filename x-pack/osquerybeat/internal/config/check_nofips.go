// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !requirefips

package config

// CheckFIPS returns nil in builds without the requirefips tag.
func CheckFIPS([]InputConfig) error {
	return nil
}
