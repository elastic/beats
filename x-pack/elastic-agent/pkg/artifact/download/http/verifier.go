// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http

import (
	"bufio"
	"bytes"
	"crypto/sha512"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"golang.org/x/crypto/openpgp"

	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
)

const (
	publicKeyURI = "https://artifacts.elastic.co/GPG-KEY-elasticsearch"
	ascSuffix    = ".asc"
	sha512Length = 128
)

// Verifier verifies a downloaded package by comparing with public ASC
// file from elastic.co website.
type Verifier struct {
	config        *artifact.Config
	client        http.Client
	pgpBytes      []byte
	allowEmptyPgp bool
}

// NewVerifier create a verifier checking downloaded package on preconfigured
// location agains a key stored on elastic.co website.
func NewVerifier(config *artifact.Config, allowEmptyPgp bool, pgp []byte) (*Verifier, error) {
	if len(pgp) == 0 && !allowEmptyPgp {
		return nil, errors.New("expecting PGP but retrieved none", errors.TypeSecurity)
	}

	client, err := config.HTTPTransportSettings.Client(
		httpcommon.WithAPMHTTPInstrumentation(),
		httpcommon.WithModRoundtripper(func(rt http.RoundTripper) http.RoundTripper {
			return withHeaders(rt, headers)
		}),
	)
	if err != nil {
		return nil, err
	}

	v := &Verifier{
		config:        config,
		client:        *client,
		allowEmptyPgp: allowEmptyPgp,
		pgpBytes:      pgp,
	}

	return v, nil
}

// Verify checks downloaded package on preconfigured
// location agains a key stored on elastic.co website.
func (v *Verifier) Verify(spec program.Spec, version string, removeOnFailure bool) (isMatch bool, err error) {
	filename, err := artifact.GetArtifactName(spec, version, v.config.OS(), v.config.Arch())
	if err != nil {
		return false, errors.New(err, "retrieving package name")
	}

	fullPath, err := artifact.GetArtifactPath(spec, version, v.config.OS(), v.config.Arch(), v.config.TargetDirectory)
	if err != nil {
		return false, errors.New(err, "retrieving package path")
	}

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

	return v.verifyAsc(spec, version)
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
		return false, fmt.Errorf("hash for '%s' not found", filename)
	}

	// compute file hash
	fileReader, err := os.OpenFile(fullPath, os.O_RDONLY, 0666)
	if err != nil {
		return false, errors.New(err, errors.TypeFilesystem, errors.M(errors.MetaKeyPath, fullPath))
	}
	defer fileReader.Close()

	// compute file hash
	hash := sha512.New()
	if _, err := io.Copy(hash, fileReader); err != nil {
		return false, err
	}
	computedHash := fmt.Sprintf("%x", hash.Sum(nil))

	return expectedHash == computedHash, nil
}

func (v *Verifier) verifyAsc(spec program.Spec, version string) (bool, error) {
	if len(v.pgpBytes) == 0 {
		// no pgp available skip verification process
		return true, nil
	}

	filename, err := artifact.GetArtifactName(spec, version, v.config.OS(), v.config.Arch())
	if err != nil {
		return false, errors.New(err, "retrieving package name")
	}

	fullPath, err := artifact.GetArtifactPath(spec, version, v.config.OS(), v.config.Arch(), v.config.TargetDirectory)
	if err != nil {
		return false, errors.New(err, "retrieving package path")
	}

	ascURI, err := v.composeURI(filename, spec.Artifact)
	if err != nil {
		return false, errors.New(err, "composing URI for fetching asc file", errors.TypeNetwork)
	}

	ascBytes, err := v.getPublicAsc(ascURI)
	if err != nil && v.allowEmptyPgp {
		// asc not available but we allow empty for dev use-case
		return true, nil
	} else if err != nil {
		return false, errors.New(err, fmt.Sprintf("fetching asc file from %s", ascURI), errors.TypeNetwork, errors.M(errors.MetaKeyURI, ascURI))
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

func (v *Verifier) composeURI(filename, artifactName string) (string, error) {
	upstream := v.config.SourceURI
	if !strings.HasPrefix(upstream, "http") && !strings.HasPrefix(upstream, "file") && !strings.HasPrefix(upstream, "/") {
		// always default to https
		upstream = fmt.Sprintf("https://%s", upstream)
	}

	// example: https://artifacts.elastic.co/downloads/beats/filebeat/filebeat-7.1.1-x86_64.rpm
	uri, err := url.Parse(upstream)
	if err != nil {
		return "", errors.New(err, "invalid upstream URI", errors.TypeNetwork, errors.M(errors.MetaKeyURI, upstream))
	}

	uri.Path = path.Join(uri.Path, artifactName, filename+ascSuffix)
	return uri.String(), nil
}

func (v *Verifier) getPublicAsc(sourceURI string) ([]byte, error) {
	resp, err := v.client.Get(sourceURI)
	if err != nil {
		return nil, errors.New(err, "failed loading public key", errors.TypeNetwork, errors.M(errors.MetaKeyURI, sourceURI))
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("call to '%s' returned unsuccessful status code: %d", sourceURI, resp.StatusCode), errors.TypeNetwork, errors.M(errors.MetaKeyURI, sourceURI))
	}

	return ioutil.ReadAll(resp.Body)
}
