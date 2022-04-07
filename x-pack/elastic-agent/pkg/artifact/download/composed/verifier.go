// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package composed

import (
	"errors"

	"github.com/hashicorp/go-multierror"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download"
)

// Verifier is a verifier with a predefined set of verifiers.
// During each verify call it tries to call the first one and on failure fallbacks to
// the next one.
// Error is returned if all of them fail.
type Verifier struct {
	vv []download.Verifier
}

// NewVerifier creates a verifier composed out of predefined set of verifiers.
// During each verify call it tries to call the first one and on failure fallbacks to
// the next one.
// Error is returned if all of them fail.
func NewVerifier(verifiers ...download.Verifier) *Verifier {
	return &Verifier{
		vv: verifiers,
	}
}

// Verify checks the package from configured source.
func (e *Verifier) Verify(spec program.Spec, version string) error {
	var err error
	var checksumMismatchErr *download.ChecksumMismatchError
	var invalidSignatureErr *download.InvalidSignatureError

	for _, v := range e.vv {
		e := v.Verify(spec, version)
		if e == nil {
			// Success
			return nil
		}

		err = multierror.Append(err, e)

		if errors.As(e, &checksumMismatchErr) || errors.As(err, &invalidSignatureErr) {
			// Stop verification chain on checksum/signature errors.
			break
		}
	}

	return err
}
