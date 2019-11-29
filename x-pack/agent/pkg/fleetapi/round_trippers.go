// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleetapi

import (
	"errors"
	"net/http"

	"github.com/elastic/beats/agent/kibana"
)

var InvalidCredentialsErr = errors.New("invalid credentials to connect to fleet")

// FleetUserAgentRoundTripper adds the Fleet user agent.
type FleetUserAgentRoundTripper struct {
	rt      http.RoundTripper
	version string
}

// RoundTrip adds the Fleet user agent string to every request.
func (r *FleetUserAgentRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return r.rt.RoundTrip(req)
}

// NewFleetUserAgentRoundTripper returns a  FleetUserAgentRoundTripper that actually wrap the
// existing UserAgentRoundTripper with a specific string.
func NewFleetUserAgentRoundTripper(wrapped http.RoundTripper, version string) http.RoundTripper {
	const name = "Beat Agent"
	return &FleetUserAgentRoundTripper{
		rt: kibana.NewUserAgentRoundTripper(wrapped, name+" v"+version),
	}
}

// FleetAuthRoundTripper allow all calls to be authenticated using the api key.
// The token is added as a header key.
type FleetAuthRoundTripper struct {
	rt     http.RoundTripper
	apiKey string
}

// RoundTrip makes all the calls to the service authenticated.
func (r *FleetAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	const key = "Authorization"
	const prefix = "ApiKey "

	req.Header.Set(key, prefix+r.apiKey)
	resp, err := r.rt.RoundTrip(req)

	if resp.StatusCode == http.StatusUnauthorized {
		defer resp.Body.Close()
		return resp, InvalidCredentialsErr
	}
	return resp, err
}

// NewFleetAuthRoundTripper wrap an existing http.RoundTripper and adds the API in the header.
func NewFleetAuthRoundTripper(
	wrapped http.RoundTripper,
	apiKey string,
) (http.RoundTripper, error) {
	if len(apiKey) == 0 {
		return nil, errors.New("empty api key received")
	}
	return &FleetAuthRoundTripper{rt: wrapped, apiKey: apiKey}, nil
}
