// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"net/http"

	uuid "github.com/satori/go.uuid"

	"github.com/elastic/beats/libbeat/common"
)

// Enroll a beat in central management, this call returns a valid access token to retrieve configurations
func (c *Client) Enroll(beatType, beatVersion, hostname string, beatUUID uuid.UUID, enrollmentToken string) (string, error) {
	params := common.MapStr{
		"type":      beatType,
		"host_name": hostname,
		"version":   beatVersion,
	}

	resp := struct {
		AccessToken string `json:"access_token"`
	}{}

	headers := http.Header{}
	headers.Set("kbn-beats-enrollment-token", enrollmentToken)

	_, err := c.request("POST", "/api/beats/agent/"+beatUUID.String(), params, headers, &resp)
	if err != nil {
		return "", err
	}

	return resp.AccessToken, err
}
