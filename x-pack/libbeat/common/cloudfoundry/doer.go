// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"crypto/tls"
	"net/http"

	"github.com/cloudfoundry-incubator/uaago"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/pkg/errors"
)

// authTokenDoer is an HTTP requester that indcludes UAA tokens at the header
type authTokenDoer struct {
	uaa          *uaago.Client
	clientID     string
	clientSecret string
	skipVerify   bool
	httpClient   *http.Client
	log          *logp.Logger
}

// NewAuthTokenDoer creates a loggregator HTTP client that uses a new UAA token at each request
func newAuthTokenDoer(uaa *uaago.Client, clientID, clientSecret string, skipVerify bool, log *logp.Logger) *authTokenDoer {
	return &authTokenDoer{
		uaa:          uaa,
		clientID:     clientID,
		clientSecret: clientSecret,
		skipVerify:   skipVerify,
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: skipVerify,
				},
			},
		},
		log: log.Named("doer"),
	}
}

// Do executes an HTTP request adding an UAA OAuth token
func (d *authTokenDoer) Do(r *http.Request) (*http.Response, error) {
	t, err := d.uaa.GetAuthToken(d.clientID, d.clientSecret, d.skipVerify)
	if err != nil {
		// The reason for writing an error here is that pushing the error upstream
		// is handled by loggregate library, which is beyond our reach.
		d.log.Errorf("error creating UAA Auth Token: %+v", err)
		return nil, errors.Wrap(err, "error retrieving UUA token")
	}
	r.Header.Set("Authorization", t)
	return d.httpClient.Do(r)
}
