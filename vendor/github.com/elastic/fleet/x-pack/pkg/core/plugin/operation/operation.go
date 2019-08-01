// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

// operation is an operation definition
// each operation needs to implement this interface in order
// to ease up rollbacks
type operation interface {
	// Name is human readable name which identifies an operation
	Name() string
	// Check  checks whether operation needs to be run
	// In case prerequisites (such as invalid cert or tweaked binary) are not met, it returns error
	// examples:
	// - Start does not need to run if process is running
	// - Fetch does not need to run if package is already present
	Check() (bool, error)
	// Run runs the operation
	Run() error
}
