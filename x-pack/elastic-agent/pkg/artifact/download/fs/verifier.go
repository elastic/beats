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
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download"

	"golang.org/x/crypto/openpgp" //nolint:staticcheck // won't update the package for now

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
)

const (
	ascSuffix    = ".asc"
	sha512Length = 128
)

// Verifier verifies a downloaded package by comparing with public ASC
// file from elastic.co website.
type Verifier struct {
	config        *artifact.Config
	pgpBytes      []byte
	allowEmptyPgp bool
	client        http.Client
}

// NewVerifier create a verifier checking downloaded package on preconfigured
// location against a key stored on elastic.co website.
func NewVerifier(config *artifact.Config, allowEmptyPgp bool, pgp []byte) (*Verifier, error) {
	if len(pgp) == 0 && !allowEmptyPgp {
		return nil, errors.New("expecting PGP but retrieved none", errors.TypeSecurity)
	}

	client, err := config.HTTPTransportSettings.Client(
		httpcommon.WithAPMHTTPInstrumentation(),
		httpcommon.WithModRoundtripper(func(rt http.RoundTripper) http.RoundTripper {
			return download.WithHeaders(rt, download.Headers)
		}),
	)
	if err != nil {
		return nil, err
	}

	v := &Verifier{
		config:        config,
		allowEmptyPgp: allowEmptyPgp,
		pgpBytes:      pgp,
		client:        *client,
	}

	return v, nil
}

// Verify checks downloaded package on preconfigured
// location against a key stored on elastic.co website.
func (v *Verifier) Verify(spec program.Spec, version string, removeOnFailure bool, pgpBytes ...string) (isMatch bool, err error) {
	filename, err := artifact.GetArtifactName(spec, version, v.config.OS(), v.config.Arch())
	if err != nil {
		return false, errors.New(err, "retrieving package name")
	}

	fullPath := filepath.Join(v.config.TargetDirectory, filename)
	defer func() {
		if removeOnFailure && (!isMatch || err != nil) {
			// remove bits so they can be redownloaded
			os.Remove(fullPath)
			os.Remove(fullPath + ".sha512")
			os.Remove(fullPath + ".asc")
		}
	}()

	if isMatch, err := v.verifyHash(filename, fullPath); !isMatch || err != nil {
		return isMatch, err
	}

	return v.verifyAsc(filename, fullPath, pgpBytes...)
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
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasSuffix(line, filename) {
			continue
		}

		if len(line) > sha512Length {
			expectedHash = strings.TrimSpace(line[:sha512Length])
		}
	}

	if expectedHash == "" {
		return false, fmt.Errorf("hash for '%s' not found in '%s'", filename, hashFilePath)
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

func (v *Verifier) verifyAsc(filename, fullPath string, pgpSources ...string) (bool, error) {
	var pgpBytes [][]byte
	if len(v.pgpBytes) > 0 {
		pgpBytes = append(pgpBytes, v.pgpBytes)
	}

	for _, check := range pgpSources {
		if len(check) == 0 {
			continue
		}
		raw, err := download.PgpBytesFromSource(check, v.client)
		if err != nil {
			return false, err
		}
		if len(raw) == 0 {
			continue
		}

		pgpBytes = append(pgpBytes, raw)
	}

	if len(pgpBytes) == 0 {
		// no pgp available skip verification process
		return true, nil
	}

	ascBytes, err := v.getPublicAsc(fullPath)
	if err != nil && v.allowEmptyPgp {
		// asc not available but we allow empty for dev use-case
		return true, nil
	} else if err != nil {
		return false, err
	}

	var lastCheckErr error
	for _, check := range pgpBytes {
		pubkeyReader := bytes.NewReader(check)
		ascReader := bytes.NewReader(ascBytes)
		fileReader, err := os.OpenFile(fullPath, os.O_RDONLY, 0666)
		if err != nil {
			lastCheckErr = err
			continue
		}
		defer fileReader.Close()

		keyring, err := openpgp.ReadArmoredKeyRing(pubkeyReader)
		if err != nil {
			lastCheckErr = err
			continue
		}
		_, err = openpgp.CheckArmoredDetachedSignature(keyring, fileReader, ascReader)
		if err != nil {
			lastCheckErr = err
			continue
		}

		return true, nil
	}

	return false, lastCheckErr
}

func (v *Verifier) getPublicAsc(fullPath string) ([]byte, error) {
	fullPath = fmt.Sprintf("%s%s", fullPath, ascSuffix)
	b, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return nil, errors.New(err, fmt.Sprintf("fetching asc file from '%s'", fullPath), errors.TypeFilesystem, errors.M(errors.MetaKeyPath, fullPath))
	}

	return b, nil
}
