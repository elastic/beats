// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package download

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
)

const (
	// PgpSourceRawPrefix prefixes raw PGP keys.
	PgpSourceRawPrefix = "pgp_raw:"
	// PgpSourceURIPrefix prefixes URI pointing to remote PGP key.
	PgpSourceURIPrefix = "pgp_uri:"
)

// Verifier is an interface verifying GPG key of a downloaded artifact
type Verifier interface {
	Verify(spec program.Spec, version string, removeOnFailure bool, pgpBytes ...string) (bool, error)
}

// PgpBytesFromSource returns clean PGP key from raw source or remote URI.
func PgpBytesFromSource(source string, client http.Client) ([]byte, error) {
	if strings.HasPrefix(source, PgpSourceRawPrefix) {
		return []byte(strings.TrimPrefix(source, PgpSourceRawPrefix)), nil
	}

	if strings.HasPrefix(source, PgpSourceURIPrefix) {
		return fetchPgpFromURI(strings.TrimPrefix(source, PgpSourceURIPrefix), client)
	}

	return nil, errors.New("unknown pgp source")
}

// CheckValidDownloadURI checks whether specified string is a valid HTTP URI.
func CheckValidDownloadURI(rawURI string) error {
	uri, err := url.Parse(rawURI)
	if err != nil {
		return err
	}

	if !strings.EqualFold(uri.Scheme, "https") {
		return fmt.Errorf("failed to check URI %q: HTTPS is required", rawURI)
	}

	return nil
}

func fetchPgpFromURI(uri string, client http.Client) ([]byte, error) {
	if err := CheckValidDownloadURI(uri); err != nil {
		return nil, err
	}

	resp, err := client.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("call to '%s' returned unsuccessful status code: %d", uri, resp.StatusCode), errors.TypeNetwork, errors.M(errors.MetaKeyURI, uri))
	}

	return ioutil.ReadAll(resp.Body)
}
