// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cloudfoundry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v8/libbeat/logp"
)

// authTokenDoer is an HTTP requester that indcludes UAA tokens at the header
type authTokenDoer struct {
	url          string
	clientID     string
	clientSecret string
	httpClient   *http.Client
	log          *logp.Logger
}

// NewAuthTokenDoer creates a loggregator HTTP client that uses a new UAA token at each request
func newAuthTokenDoer(url string, clientID, clientSecret string, httpClient *http.Client, log *logp.Logger) *authTokenDoer {
	return &authTokenDoer{
		url:          url,
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   httpClient,
		log:          log.Named("doer"),
	}
}

// Do executes an HTTP request adding an UAA OAuth token
func (d *authTokenDoer) Do(r *http.Request) (*http.Response, error) {
	t, err := d.getAuthToken(d.clientID, d.clientSecret)
	if err != nil {
		// The reason for writing an error here is that pushing the error upstream
		// is handled by loggregate library, which is beyond our reach.
		d.log.Errorf("error creating UAA Auth Token: %+v", err)
		return nil, errors.Wrap(err, "error retrieving UUA token")
	}
	r.Header.Set("Authorization", t)
	return d.httpClient.Do(r)
}

func (d *authTokenDoer) getAuthToken(username, password string) (string, error) {
	token, _, err := d.getAuthTokenWithExpiresIn(username, password)
	return token, err
}

func (d *authTokenDoer) getAuthTokenWithExpiresIn(username, password string) (string, int, error) {
	data := url.Values{
		"client_id":  {username},
		"grant_type": {"client_credentials"},
	}

	request, err := http.NewRequest("POST", fmt.Sprintf("%s/oauth/token", d.url), strings.NewReader(data.Encode()))
	if err != nil {
		return "", -1, err
	}
	request.SetBasicAuth(username, password)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := d.httpClient.Do(request)
	if err != nil {
		return "", -1, err
	}

	if resp.StatusCode != http.StatusOK {
		return "", -1, fmt.Errorf("received a status code %v", resp.Status)
	}
	defer resp.Body.Close()

	jsonData := make(map[string]interface{})
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&jsonData)
	if err != nil {
		return "", -1, err
	}

	expiresIn := 0
	if value, ok := jsonData["expires_in"]; ok {
		asFloat, err := strconv.ParseFloat(fmt.Sprintf("%f", value), 64)
		if err != nil {
			return "", -1, err
		}
		expiresIn = int(asFloat)
	}

	return fmt.Sprintf("%s %s", jsonData["token_type"], jsonData["access_token"]), expiresIn, nil
}
