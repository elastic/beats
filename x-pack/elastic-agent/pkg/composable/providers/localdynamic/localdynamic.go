// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package localdynamic

import (
	"fmt"
	"strconv"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
)

func init() {
	composable.Providers.AddDynamicProvider("local_dynamic", DynamicProviderBuilder)
}

type dynamicItem struct {
	Mapping    map[string]interface{}   `config:"vars"`
	Processors []map[string]interface{} `config:"processors"`
}

type dynamicProvider struct {
	Items []dynamicItem `config:"items"`
}

// Run runs the environment context provider.
func (c *dynamicProvider) Run(comm composable.DynamicProviderComm) error {
	for i, item := range c.Items {
		if err := comm.AddOrUpdate(strconv.Itoa(i), item.Mapping, item.Processors); err != nil {
			return errors.New(err, fmt.Sprintf("failed to add mapping for index %d", i), errors.TypeUnexpected)
		}
	}
	return nil
}

// DynamicProviderBuilder builds the dynamic provider.
func DynamicProviderBuilder(c *config.Config) (composable.DynamicProvider, error) {
	p := &dynamicProvider{}
	if c != nil {
		err := c.Unpack(p)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack vars: %s", err)
		}
	}
	if p.Items == nil {
		p.Items = []dynamicItem{}
	}
	return p, nil
}
