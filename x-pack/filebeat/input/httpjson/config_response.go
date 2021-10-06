// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"fmt"
	"strings"
)

const (
	splitTypeArr    = "array"
	splitTypeMap    = "map"
	splitTypeString = "string"
)

type responseConfig struct {
	DecodeAs                string           `config:"decode_as"`
	RequestBodyOnPagination bool             `config:"request_body_on_pagination"`
	Transforms              transformsConfig `config:"transforms"`
	Pagination              transformsConfig `config:"pagination"`
	Split                   *splitConfig     `config:"split"`
}

type splitConfig struct {
	Target           string           `config:"target" validation:"required"`
	Type             string           `config:"type"`
	Transforms       transformsConfig `config:"transforms"`
	Split            *splitConfig     `config:"split"`
	KeepParent       bool             `config:"keep_parent"`
	KeyField         string           `config:"key_field"`
	DelimiterString  string           `config:"delimiter"`
	IgnoreEmptyValue bool             `config:"ignore_empty_value"`
}

func (c *responseConfig) Validate() error {
	if _, err := newBasicTransformsFromConfig(c.Transforms, responseNamespace, nil); err != nil {
		return err
	}
	if _, err := newBasicTransformsFromConfig(c.Pagination, paginationNamespace, nil); err != nil {
		return err
	}
	if c.DecodeAs != "" {
		if _, found := registeredDecoders[c.DecodeAs]; !found {
			return fmt.Errorf("decoder not found for contentType: %v", c.DecodeAs)
		}
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
	case splitTypeString:
		if c.DelimiterString == "" {
			return fmt.Errorf("delimiter required for split type %s", splitTypeString)
		}
	default:
		return fmt.Errorf("invalid split type: %s", c.Type)
	}

	if _, err := newSplitResponse(c, nil); err != nil {
		return err
	}

	return nil
}
