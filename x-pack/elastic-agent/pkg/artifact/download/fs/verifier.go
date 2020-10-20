// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fs

import (
	"bufio"
	"bytes"
	"crypto/sha512"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/crypto/openpgp"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
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

	return v, nil
}

// Verify checks downloaded package on preconfigured
// location agains a key stored on elastic.co website.
func (v *Verifier) Verify(programName, version string) (bool, error) {
	filename, err := artifact.GetArtifactName(programName, version, v.config.OS(), v.config.Arch())
	if err != nil {
		return false, errors.New(err, "retrieving package name")
	}

	fullPath := filepath.Join(v.config.TargetDirectory, filename)

	isMatch, err := v.verifyHash(filename, fullPath)
	if !isMatch || err != nil {
		// remove bits so they can be redownloaded
		os.Remove(fullPath)
		os.Remove(fullPath + ".sha512")
	}

	return isMatch, err
}

func (v *Verifier) verifyHash(filename, fullPath string) (bool, error) {
	hashFilePath := fullPath + ".sha512"
	hashFileHandler, err := os.Open(hashFilePath)
	if err != nil {
		return false, err
	}
	defer hashFileHandler.Close()

	// get hash
	// content of a file is in following format
	// hash  filename
	var expectedHash string
	scanner := bufio.NewScanner(hashFileHandler)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasSuffix(line, filename) {
			continue
		}

		expectedHash = strings.TrimSpace(strings.TrimSuffix(line, filename))
	}

	if expectedHash == "" {
		return false, fmt.Errorf("hash for '%s' not found", filename)
	}

	// compute file hash
	fileReader, err := os.OpenFile(fullPath, os.O_RDONLY, 0666)
	if err != nil {
		return false, errors.New(err, errors.TypeFilesystem, errors.M(errors.MetaKeyPath, fullPath))
	}
	defer fileReader.Close()

	hash := sha512.New()
	if _, err := io.Copy(hash, fileReader); err != nil {
		return false, err
	}
	computedHash := fmt.Sprintf("%x", hash.Sum(nil))

	return expectedHash == computedHash, nil
}

func (v *Verifier) verifyAsc(filename, fullPath string) (bool, error) {
	var err error
	var pgpBytesLoader sync.Once

	pgpBytesLoader.Do(func() {
		err = v.loadPGP(v.config.PgpFile)
	})

	if err != nil {
		return false, errors.New(err, "loading PGP")
	}

	ascBytes, err := v.getPublicAsc(filename)
	if err != nil {
		return false, err
	}

	pubkeyReader := bytes.NewReader(v.pgpBytes)
	ascReader := bytes.NewReader(ascBytes)
	fileReader, err := os.OpenFile(fullPath, os.O_RDONLY, 0666)
	if err != nil {
		return false, errors.New(err, errors.TypeFilesystem, errors.M(errors.MetaKeyPath, fullPath))
	}
	defer fileReader.Close()

	keyring, err := openpgp.ReadArmoredKeyRing(pubkeyReader)
	if err != nil {
		return false, errors.New(err, "read armored key ring", errors.TypeSecurity)
	}
	_, err = openpgp.CheckArmoredDetachedSignature(keyring, fileReader, ascReader)
	if err != nil {
		return false, errors.New(err, "check detached signature", errors.TypeSecurity)
	}

	return true, nil
}

func (v *Verifier) getPublicAsc(filename string) ([]byte, error) {
	ascFile := fmt.Sprintf("%s%s", filename, ascSuffix)
	fullPath := filepath.Join(paths.Home(), "downloads", ascFile)

	b, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return nil, errors.New(err, fmt.Sprintf("fetching asc file from '%s'", fullPath), errors.TypeFilesystem, errors.M(errors.MetaKeyPath, fullPath))
	}

	return b, nil
}

func (v *Verifier) loadPGP(file string) error {
	var err error

	if file == "" {
		return errors.New("pgp file not specified for verifier", errors.TypeConfig)
	}

	v.pgpBytes, err = ioutil.ReadFile(file)
	if err != nil {
		return errors.New(err, errors.TypeFilesystem)
	}

	return nil
}
