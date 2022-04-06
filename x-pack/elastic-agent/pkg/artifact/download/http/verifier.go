// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/artifact/download"
)

const (
	ascSuffix = ".asc"
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
// location against a key stored on elastic.co website.
func (v *Verifier) Verify(spec program.Spec, version string) error {
	fullPath, err := artifact.GetArtifactPath(spec, version, v.config.OS(), v.config.Arch(), v.config.TargetDirectory)
	if err != nil {
		return errors.New(err, "retrieving package path")
	}

	if err = download.VerifySHA512Hash(fullPath); err != nil {
		var checksumMismatchErr *download.ChecksumMismatchError
		if errors.As(err, &checksumMismatchErr) {
			os.Remove(fullPath)
			os.Remove(fullPath + ".sha512")
		}
		return err
	}

	if err = v.verifyAsc(spec, version); err != nil {
		var invalidSignatureErr *download.InvalidSignatureError
		if errors.As(err, &invalidSignatureErr) {
			os.Remove(fullPath + ".asc")
		}
		return err
	}

	return nil
}

func (v *Verifier) verifyAsc(spec program.Spec, version string) error {
	if len(v.pgpBytes) == 0 {
		// no pgp available skip verification process
		return nil
	}

	filename, err := artifact.GetArtifactName(spec, version, v.config.OS(), v.config.Arch())
	if err != nil {
		return errors.New(err, "retrieving package name")
	}

	fullPath, err := artifact.GetArtifactPath(spec, version, v.config.OS(), v.config.Arch(), v.config.TargetDirectory)
	if err != nil {
		return errors.New(err, "retrieving package path")
	}

	ascURI, err := v.composeURI(filename, spec.Artifact)
	if err != nil {
		return errors.New(err, "composing URI for fetching asc file", errors.TypeNetwork)
	}

	ascBytes, err := v.getPublicAsc(ascURI)
	if err != nil && v.allowEmptyPgp {
		// asc not available but we allow empty for dev use-case
		return nil
	} else if err != nil {
		return errors.New(err, fmt.Sprintf("fetching asc file from %s", ascURI), errors.TypeNetwork, errors.M(errors.MetaKeyURI, ascURI))
	}

	return download.VerifyGPGSignature(fullPath, ascBytes, v.pgpBytes)
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
