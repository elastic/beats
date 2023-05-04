// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package composed

import (
	"github.com/hashicorp/go-multierror"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
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
func (e *Verifier) Verify(spec program.Spec, version string, removeOnFailure bool, pgpBytes ...string) (bool, error) {
	var err error

	for i, v := range e.vv {
		isLast := (i + 1) == len(e.vv)
		b, e := v.Verify(spec, version, isLast && removeOnFailure, pgpBytes...)
		if e == nil {
			return b, nil
		}

		err = multierror.Append(err, e)
	}

	return false, err
}

// Reload reloads config
func (e *Verifier) Reload(c *artifact.Config) error {
	for _, v := range e.vv {
		reloadable, ok := v.(download.Reloader)
		if !ok {
			continue
		}

		if err := reloadable.Reload(c); err != nil {
			return errors.New(err, "failed reloading artifact config for composed verifier")
		}
	}
	return nil
}
