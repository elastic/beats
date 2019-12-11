// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
)

type enrollResponse struct {
	BaseResponse
	AccessToken string `json:"item"`
}

func (e *enrollResponse) Validate() error {
	if !e.Success || len(e.AccessToken) == 0 {
		return errors.New("empty access_token")
	}
	return nil
}

// Enroll a beat in central management, this call returns a valid access token to retrieve
// configurations
func (c *Client) Enroll(
	beatType, beatName, beatVersion, hostname string,
	beatUUID uuid.UUID,
	enrollmentToken string,
) (string, error) {
	params := common.MapStr{
		"type":      beatType,
		"name":      beatName,
		"version":   beatVersion,
		"host_name": hostname,
	}

	resp := enrollResponse{}

	headers := http.Header{}
	headers.Set("kbn-beats-enrollment-token", enrollmentToken)

	_, err := c.request("POST", "/api/beats/agent/"+beatUUID.String(), params, headers, &resp)
	if err != nil {
		return "", err
	}

	if err := resp.Validate(); err != nil {
		return "", err
	}

	return resp.AccessToken, err
}
