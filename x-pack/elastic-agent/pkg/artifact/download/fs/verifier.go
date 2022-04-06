// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download"
)

const (
	ascSuffix = ".asc"
)

// Verifier verifies an artifact's GPG signature as read from the filesystem.
// The signature is validated against Elastic's public GPG key that is
// embedded into Elastic Agent.
type Verifier struct {
	config        *artifact.Config
	pgpBytes      []byte
	allowEmptyPgp bool
}

// NewVerifier creates a verifier checking downloaded package on preconfigured
// location against a key stored on elastic.co website.
func NewVerifier(config *artifact.Config, allowEmptyPgp bool, pgp []byte) (*Verifier, error) {
	if len(pgp) == 0 && !allowEmptyPgp {
		return nil, errors.New("expecting PGP but retrieved none", errors.TypeSecurity)
	}

	v := &Verifier{
		config:        config,
		allowEmptyPgp: allowEmptyPgp,
		pgpBytes:      pgp,
	}

	return v, nil
}

// Verify checks downloaded package on preconfigured
// location against a key stored on elastic.co website.
func (v *Verifier) Verify(spec program.Spec, version string) error {
	filename, err := artifact.GetArtifactName(spec, version, v.config.OS(), v.config.Arch())
	if err != nil {
		return errors.New(err, "retrieving package name")
	}

	fullPath := filepath.Join(v.config.TargetDirectory, filename)

	if err = download.VerifySHA512Hash(fullPath); err != nil {
		var checksumMismatchErr *download.ChecksumMismatchError
		if errors.As(err, &checksumMismatchErr) {
			os.Remove(fullPath)
			os.Remove(fullPath + ".sha512")
		}
		return err
	}

	if err = v.verifyAsc(fullPath); err != nil {
		var invalidSignatureErr *download.InvalidSignatureError
		if errors.As(err, &invalidSignatureErr) {
			os.Remove(fullPath + ".asc")
		}
		return err
	}

	return nil
}

func (v *Verifier) verifyAsc(fullPath string) error {
	if len(v.pgpBytes) == 0 {
		// no pgp available skip verification process
		return nil
	}

	ascBytes, err := v.getPublicAsc(fullPath)
	if err != nil && v.allowEmptyPgp {
		// asc not available but we allow empty for dev use-case
		return nil
	} else if err != nil {
		return err
	}

	return download.VerifyGPGSignature(fullPath, ascBytes, v.pgpBytes)
}

func (v *Verifier) getPublicAsc(fullPath string) ([]byte, error) {
	fullPath = fmt.Sprintf("%s%s", fullPath, ascSuffix)
	b, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return nil, errors.New(err, fmt.Sprintf("fetching asc file from '%s'", fullPath), errors.TypeFilesystem, errors.M(errors.MetaKeyPath, fullPath))
	}

	return b, nil
}
