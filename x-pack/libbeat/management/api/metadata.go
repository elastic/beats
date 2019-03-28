// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"fmt"
	"net/http"

	"github.com/elastic/beats/libbeat/common"
)

// UpdateMetadata sends the metadata assigned to the beat
func (c *AuthClient) UpdateMetadata(meta common.MapStr) error {
	params := common.MapStr{
		"metadata": meta,
	}

	url := fmt.Sprintf("/api/beats/agent/%s", c.BeatUUID)
	resp := make(map[string]interface{})
	statusCode, err := c.Client.request("PUT", url, params, c.headers(), &resp)
	if statusCode == http.StatusNotFound {
		return errConfigurationNotFound
	}

	return err
}
