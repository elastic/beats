// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package download

// Verifier is an interface verifying GPG key of a downloaded artifact
type Verifier interface {
	Verify(programName, version string) (bool, error)
}
