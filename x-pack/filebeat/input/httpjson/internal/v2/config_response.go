// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"fmt"
	"strings"
)

const (
	splitTypeArr = "array"
	splitTypeMap = "map"
)

type responseConfig struct {
	Transforms transformsConfig `config:"transforms"`
	Pagination transformsConfig `config:"pagination"`
	Split      *splitConfig     `config:"split"`
}

type splitConfig struct {
	Target     string           `config:"target" validation:"required"`
	Type       string           `config:"type"`
	Transforms transformsConfig `config:"transforms"`
	Split      *splitConfig     `config:"split"`
	KeepParent bool             `config:"keep_parent"`
	KeyField   string           `config:"key_field"`
}

func (c *responseConfig) Validate() error {
	if _, err := newBasicTransformsFromConfig(c.Transforms, responseNamespace, nil); err != nil {
		return err
	}
	if _, err := newBasicTransformsFromConfig(c.Pagination, paginationNamespace, nil); err != nil {
		return err
	}
	return nil
}

func (c *splitConfig) Validate() error {
	if _, err := newBasicTransformsFromConfig(c.Transforms, responseNamespace, nil); err != nil {
		return err
	}

	c.Type = strings.ToLower(c.Type)
	switch c.Type {
	case "", splitTypeArr:
		if c.KeyField != "" {
			return fmt.Errorf("key_field can only be used with a %s split type", splitTypeMap)
		}
	case splitTypeMap:
	default:
		return fmt.Errorf("invalid split type: %s", c.Type)
	}

	if _, err := newSplitResponse(c, nil); err != nil {
		return err
	}

	return nil
}
