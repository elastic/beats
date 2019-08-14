// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"fmt"
	"net/http"
)

// CreateEnrollmentToken talks to Kibana API and generates an enrollment token
func (c *Client) CreateEnrollmentToken() (string, error) {
	headers := http.Header{}

	resp := struct {
		Results []struct {
			Token string `json:"item"`
		} `json:"results"`
	}{}

	_, err := c.request("POST", "/api/beats/enrollment_tokens", nil, headers, &resp)
	if err != nil {
		return "", err
	}

	if tokensCount := len(resp.Results); tokensCount != 1 {
		return "", fmt.Errorf("Unexpected number of tokens, got %d, only one expected", tokensCount)
	}

	return resp.Results[0].Token, nil
}
