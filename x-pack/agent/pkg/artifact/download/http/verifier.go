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

	"golang.org/x/crypto/openpgp"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/artifact"
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
		return nil, errors.New(err, "loading PGP")
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

	fullPath, err := artifact.GetArtifactPath(programName, version, v.config.OS(), v.config.Arch(), v.config.TargetDirectory)
	if err != nil {
		return false, errors.New(err, "retrieving package path")
	}

	ascURI, err := v.composeURI(programName, filename)
	if err != nil {
		return false, errors.New(err, "composing URI for fetching asc file", errors.TypeNetwork)
	}

	ascBytes, err := v.getPublicAsc(ascURI)
	if err != nil {
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

func (v *Verifier) composeURI(programName, filename string) (string, error) {
	upstream := v.config.BeatsSourceURI
	if !strings.HasPrefix(upstream, "http") && !strings.HasPrefix(upstream, "file") && !strings.HasPrefix(upstream, "/") {
		// always default to https
		upstream = fmt.Sprintf("https://%s", upstream)
	}

	// example: https://artifacts.elastic.co/downloads/beats/filebeat/filebeat-7.1.1-x86_64.rpm
	uri, err := url.Parse(upstream)
	if err != nil {
		return "", errors.New(err, "invalid upstream URI", errors.TypeNetwork, errors.M(errors.MetaKeyURI, upstream))
	}

	uri.Path = path.Join(uri.Path, programName, filename+ascSuffix)
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

func (v *Verifier) loadPGP(file string) error {
	var err error

	if file == "" {
		v.pgpBytes, err = v.loadPGPFromWeb()
		return err
	}

	v.pgpBytes, err = ioutil.ReadFile(file)
	if err != nil {
		return errors.New(err, errors.TypeFilesystem, errors.M(errors.MetaKeyPath, file))
	}

	return nil
}

func (v *Verifier) loadPGPFromWeb() ([]byte, error) {
	resp, err := v.client.Get(publicKeyURI)
	if err != nil {
		return nil, errors.New(err, "failed loading public key", errors.TypeNetwork, errors.M(errors.MetaKeyURI, publicKeyURI))
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("call to '%s' returned unsuccessful status code: %d", publicKeyURI, resp.StatusCode), errors.TypeNetwork, errors.M(errors.MetaKeyURI, publicKeyURI))
	}

	return ioutil.ReadAll(resp.Body)
}
