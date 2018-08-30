// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"net/http"

	uuid "github.com/satori/go.uuid"

	"github.com/elastic/beats/libbeat/common"
)

// ConfigBlock stores a piece of config from central management
type ConfigBlock struct {
	Type string `json:"type"`
	Raw  string `json:"block_yml"`
}

// Config returns a common.Config object holding the config from this block
func (c *ConfigBlock) Config() (*common.Config, error) {
	return common.NewConfigWithYAML([]byte(c.Raw), "")
}

// Configuration retrieves the list of configuration blocks from Kibana
func (c *Client) Configuration(accessToken string, beatUUID uuid.UUID) ([]*ConfigBlock, error) {
	headers := http.Header{}
	headers.Set("kbn-beats-access-token", accessToken)

	resp := struct {
		ConfigBlocks []*ConfigBlock `json:"configuration_blocks"`
	}{}
	_, err := c.request("GET", "/api/beats/agent/"+beatUUID.String()+"/configuration", nil, headers, &resp)
	if err != nil {
		return nil, err
	}

	return resp.ConfigBlocks, err
}
