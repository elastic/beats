// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"net/http"
	"reflect"

	"github.com/elastic/beats/libbeat/common/reload"

	uuid "github.com/satori/go.uuid"

	"github.com/elastic/beats/libbeat/common"
)

// ConfigBlock stores a piece of config from central management
type ConfigBlock struct {
	Raw map[string]interface{}
}

// ConfigBlocksWithType is a list of config blocks with the same type
type ConfigBlocksWithType struct {
	Type   string
	Blocks []*ConfigBlock
}

// ConfigBlocks holds a list of type + configs objects
type ConfigBlocks []ConfigBlocksWithType

// Config returns a common.Config object holding the config from this block
func (c *ConfigBlock) Config() (*common.Config, error) {
	return common.NewConfigFrom(c.Raw)
}

// ConfigWithMeta returns a reload.ConfigWithMeta object holding the config from this block, meta will be nil
func (c *ConfigBlock) ConfigWithMeta() (*reload.ConfigWithMeta, error) {
	config, err := c.Config()
	if err != nil {
		return nil, err
	}
	return &reload.ConfigWithMeta{
		Config: config,
	}, nil
}

// Configuration retrieves the list of configuration blocks from Kibana
func (c *Client) Configuration(accessToken string, beatUUID uuid.UUID) (ConfigBlocks, error) {
	headers := http.Header{}
	headers.Set("kbn-beats-access-token", accessToken)

	resp := struct {
		ConfigBlocks []*struct {
			Type string                 `json:"type"`
			Raw  map[string]interface{} `json:"config"`
		} `json:"configuration_blocks"`
	}{}
	_, err := c.request("GET", "/api/beats/agent/"+beatUUID.String()+"/configuration", nil, headers, &resp)
	if err != nil {
		return nil, err
	}

	blocks := map[string][]*ConfigBlock{}
	for _, block := range resp.ConfigBlocks {
		blocks[block.Type] = append(blocks[block.Type], &ConfigBlock{Raw: block.Raw})
	}

	res := ConfigBlocks{}
	for t, b := range blocks {
		res = append(res, ConfigBlocksWithType{Type: t, Blocks: b})
	}

	return res, nil
}

// ConfigBlocksEqual returns true if the given config blocks are equal, false if not
func ConfigBlocksEqual(a, b ConfigBlocks) bool {
	if len(a) != len(b) {
		return false
	}

	if len(a) == 0 {
		return true
	}

	return reflect.DeepEqual(a, b)
}
