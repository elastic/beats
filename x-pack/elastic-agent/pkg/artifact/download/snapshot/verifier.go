// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package snapshot

import (
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download/http"
)

// Verifier is responsible for verifying downloaded artifacts
type Verifier struct {
	verifier        download.Verifier
	versionOverride string
}

// NewVerifier creates a downloader which first checks local directory
// and then fallbacks to remote if configured.
func NewVerifier(config *artifact.Config, allowEmptyPgp bool, pgp []byte, versionOverride string) (download.Verifier, error) {
	cfg, err := snapshotConfig(config, versionOverride)
	if err != nil {
		return nil, err
	}

	v, err := http.NewVerifier(cfg, allowEmptyPgp, pgp)
	if err != nil {
		return nil, errors.New(err, "failed to create snapshot verifier")
	}

	return &Verifier{
		verifier:        v,
		versionOverride: versionOverride,
	}, nil
}

// Verify checks the package from configured source.
func (e *Verifier) Verify(spec program.Spec, version string, removeOnFailure bool, pgpBytes ...string) (bool, error) {
	return e.verifier.Verify(spec, version, removeOnFailure, pgpBytes...)
}

// Reload reloads config
func (e *Verifier) Reload(c *artifact.Config) error {
	reloader, ok := e.verifier.(artifact.ConfigReloader)
	if !ok {
		return nil
	}

	cfg, err := snapshotConfig(c, e.versionOverride)
	if err != nil {
		return errors.New(err, "snapshot.downloader: failed to generate snapshot config")
	}

	return reloader.Reload(cfg)

}
