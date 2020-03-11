// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build !linux
// +build !darwin

package app

func getUserGroup(spec ProcessSpec) (int, int, error) {
	return 0, 0, nil
}
