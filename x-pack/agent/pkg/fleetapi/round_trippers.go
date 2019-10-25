// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fleetapi

import (
	"errors"
	"net/http"

	"github.com/elastic/beats/agent/kibana"
)

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

// FleetAccessTokenRoundTripper allow all calls to be authenticated using the accessToken.
// The token is added as a header key.
type FleetAccessTokenRoundTripper struct {
	rt          http.RoundTripper
	accessToken string
}

// RoundTrip makes all the calls to the service authenticated.
func (r *FleetAccessTokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	const key = "kbn-fleet-access-token"
	req.Header.Set(key, r.accessToken)
	return r.rt.RoundTrip(req)
}

// NewFleetAccessTokenRoundTripper wrap an existing http.RoundTripper and adds the accessToken in the header.
func NewFleetAccessTokenRoundTripper(
	wrapped http.RoundTripper,
	accessToken string,
) (http.RoundTripper, error) {
	if len(accessToken) == 0 {
		return nil, errors.New("empty access token received")
	}
	return &FleetAccessTokenRoundTripper{rt: wrapped, accessToken: accessToken}, nil
}
