// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/crypto/openpgp"

	"github.com/elastic/fleet/x-pack/pkg/artifact"
)

const (
	publicKeyURI = "https://artifacts.elastic.co/GPG-KEY-elasticsearch"
	ascSuffix    = ".asc"
)

// Verifier verifies a downloaded package by comparing with public ASC
// file from elastic.co website.
type Verifier struct {
	config   *artifact.Config
	client   http.Client
	pgpBytes []byte
}

// NewVerifier create a verifier checking downloaded package on preconfigured
// location agains a key stored on elastic.co website.
func NewVerifier(config *artifact.Config) (*Verifier, error) {
	client := http.Client{Timeout: config.Timeout}
	rtt := withHeaders(client.Transport, headers)
	client.Transport = rtt
	v := &Verifier{
		config: config,
		client: client,
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

	fullPath, err := artifact.GetArtifactPath(programName, version, v.config.OS(), v.config.Arch(), v.config.TargetDirectory)
	if err != nil {
		return false, errors.Wrap(err, "retrieving package path")
	}

	ascURI, err := v.composeURI(programName, filename)
	if err != nil {
		return false, errors.Wrap(err, "composing URI for fetching asc file")
	}

	ascBytes, err := v.getPublicAsc(ascURI)
	if err != nil {
		return false, errors.Wrapf(err, "fetching asc file from %s", ascURI)
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

func (v *Verifier) composeURI(programName, filename string) (string, error) {
	upstream := v.config.BeatsSourceURI
	if !strings.HasPrefix(upstream, "http") && !strings.HasPrefix(upstream, "file") && !strings.HasPrefix(upstream, "/") {
		// always default to https
		upstream = fmt.Sprintf("https://%s", upstream)
	}

	// example: https://artifacts.elastic.co/downloads/beats/filebeat/filebeat-7.1.1-x86_64.rpm
	uri, err := url.Parse(upstream)
	if err != nil {
		return "", errors.Wrap(err, "invalid upstream URI")
	}

	uri.Path = path.Join(uri.Path, programName, filename+ascSuffix)
	return uri.String(), nil
}

func (v *Verifier) getPublicAsc(sourceURI string) ([]byte, error) {
	resp, err := v.client.Get(sourceURI)
	if err != nil {
		return nil, errors.Wrap(err, "generating package name failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("call to '%s' returned unsuccessful status code: %d", sourceURI, resp.StatusCode)
	}

	return ioutil.ReadAll(resp.Body)
}

func (v *Verifier) loadPGP(file string) error {
	var err error

	if file == "" {
		v.pgpBytes, err = v.loadPGPFromWeb()
		return err
	}

	v.pgpBytes, err = ioutil.ReadFile(file)
	return err
}

func (v *Verifier) loadPGPFromWeb() ([]byte, error) {
	resp, err := v.client.Get(publicKeyURI)
	if err != nil {
		return nil, errors.Wrap(err, "generating package name failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("call to '%s' returned unsuccessful status code: %d", publicKeyURI, resp.StatusCode)
	}

	return ioutil.ReadAll(resp.Body)
}
