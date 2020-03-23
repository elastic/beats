// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package snapshot

import (
	"fmt"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact/download"
)

// Verifier tweaks embedded verifier to handle Snapshots only.
type Verifier struct {
	embeddedVerifier download.Verifier
}

// NewVerifier creates a snapshot verifier composed out of predefined verifier.
func NewVerifier(verifier download.Verifier) *Verifier {
	return &Verifier{
		embeddedVerifier: verifier,
	}
}

// Verify checks the package from configured source.
func (e *Verifier) Verify(programName, version string) (bool, error) {
	version = fmt.Sprintf("%s-SNAPSHOT", version)
	return e.embeddedVerifier.Verify(programName, version)
}
