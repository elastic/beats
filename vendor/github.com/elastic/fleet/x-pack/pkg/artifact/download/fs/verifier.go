// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fs

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/elastic/fleet/x-pack/pkg/artifact"
	"github.com/pkg/errors"
	"golang.org/x/crypto/openpgp"
)

const (
	ascSuffix = ".asc"
)

// Verifier verifies a downloaded package by comparing with public ASC
// file from elastic.co website.
type Verifier struct {
	config   *artifact.Config
	pgpBytes []byte
}

// NewVerifier create a verifier checking downloaded package on preconfigured
// location agains a key stored on elastic.co website.
func NewVerifier(config *artifact.Config) (*Verifier, error) {
	v := &Verifier{
		config: config,
	}

	if err := v.loadPGP(config.PgpFile); err != nil {
		return nil, errors.Wrap(err, "loading PGP")
	}

	return v, nil
}

// Verify checks downloaded package on preconfigured
// location agains a key stored on elastic.co website.
func (v *Verifier) Verify(programName, version string) (bool, error) {
	filename, err := artifact.GetArtifactName(programName, version, v.config.OS(), v.config.Arch())
	if err != nil {
		return false, errors.Wrap(err, "retrieving package name")
	}

	fullPath := filepath.Join(v.config.TargetDirectory, filename)

	ascBytes, err := v.getPublicAsc(filename)
	if err != nil {
		return false, err
	}

	pubkeyReader := bytes.NewReader(v.pgpBytes)
	ascReader := bytes.NewReader(ascBytes)
	fileReader, err := os.OpenFile(fullPath, os.O_RDONLY, 0666)
	if err != nil {
		return false, err
	}
	defer fileReader.Close()

	keyring, err := openpgp.ReadArmoredKeyRing(pubkeyReader)
	if err != nil {
		return false, errors.Wrap(err, "read armored key ring")
	}
	_, err = openpgp.CheckArmoredDetachedSignature(keyring, fileReader, ascReader)
	if err != nil {
		return false, errors.Wrap(err, "check detached signature")
	}

	return true, nil
}

func (v *Verifier) getPublicAsc(filename string) ([]byte, error) {
	ascFile := fmt.Sprintf("%s%s", filename, ascSuffix)
	fullPath := filepath.Join(beatsSubfolder, ascFile)

	b, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return nil, errors.Wrapf(err, "fetching asc file from '%s'", fullPath)
	}

	return b, nil
}

func (v *Verifier) loadPGP(file string) error {
	var err error

	if file == "" {
		return errors.New("pgp file not specified for verifier")
	}

	v.pgpBytes, err = ioutil.ReadFile(file)
	return err
}
