// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/mitchellh/hashstructure"
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common/reload"

	"github.com/elastic/beats/v7/libbeat/common"
)

var errConfigurationNotFound = errors.New("no configuration found, you need to enroll your Beat")

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

type configResponse struct {
	Type string
	Raw  map[string]interface{}
}

func (c *configResponse) UnmarshalJSON(b []byte) error {
	var resp = struct {
		Type string                 `json:"type"`
		Raw  map[string]interface{} `json:"config"`
	}{}

	if err := json.Unmarshal(b, &resp); err != nil {
		return err
	}

	converter := selectConverter(resp.Type)
	newMap, err := converter(resp.Raw)
	if err != nil {
		return err
	}
	*c = configResponse{
		Type: resp.Type,
		Raw:  newMap,
	}
	return nil
}

// Configuration retrieves the list of configuration blocks from Kibana
func (c *AuthClient) Configuration() (ConfigBlocks, error) {
	resp := struct {
		BaseResponse
		ConfigBlocks []*configResponse `json:"list"`
	}{}
	url := fmt.Sprintf("/api/beats/agent/%s/configuration", c.BeatUUID)
	statusCode, err := c.Client.request("GET", url, nil, c.headers(), &resp)
	if statusCode == http.StatusNotFound {
		return nil, errConfigurationNotFound
	}

	if err != nil {
		return nil, err
	}

	blocks := map[string][]*ConfigBlock{}
	for _, block := range resp.ConfigBlocks {
		blocks[block.Type] = append(blocks[block.Type], &ConfigBlock{Raw: block.Raw})
	}

	// keep the ordering consistent while grouping the items.
	keys := make([]string, 0, len(blocks))
	for k := range blocks {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	res := ConfigBlocks{}
	for _, t := range keys {
		b := blocks[t]
		res = append(res, ConfigBlocksWithType{Type: t, Blocks: b})
	}

	return res, nil
}

// ConfigBlocksEqual returns true if the given config blocks are equal, false if not
func ConfigBlocksEqual(a, b ConfigBlocks) (bool, error) {
	// If there is an errors when hashing the config blocks its because the format changed.
	aHash, err := hashstructure.Hash(a, nil)
	if err != nil {
		return false, errors.Wrap(err, "could not hash config blocks")
	}

	bHash, err := hashstructure.Hash(b, nil)
	if err != nil {
		return false, errors.Wrap(err, "could not hash config blocks")
	}

	return aHash == bHash, nil
}

// IsConfigurationNotFound returns true if the configuration was not found.
func IsConfigurationNotFound(err error) bool {
	return err == errConfigurationNotFound
}
