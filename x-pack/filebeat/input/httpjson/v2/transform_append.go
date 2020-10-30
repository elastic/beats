// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
)

const appendName = "append"

type appendConfig struct {
	Target  string    `config:"target"`
	Value   *valueTpl `config:"value"`
	Default string    `config:"default"`
}

type appendt struct {
	targetInfo   targetInfo
	value        *valueTpl
	defaultValue string

	runFunc func(ctx transformContext, transformable *transformable, key, val string) error
}

func (appendt) transformName() string { return appendName }

func newAppendRequest(cfg *common.Config) (transform, error) {
	append, err := newAppend(cfg)
	if err != nil {
		return nil, err
	}

	switch append.targetInfo.Type {
	case targetBody:
		append.runFunc = appendBody
	case targetHeader:
		append.runFunc = appendHeader
	case targetURLParams:
		append.runFunc = appendURLParams
	default:
		return nil, fmt.Errorf("invalid target type: %s", append.targetInfo.Type)
	}

	return &append, nil
}

func newAppendResponse(cfg *common.Config) (transform, error) {
	append, err := newAppend(cfg)
	if err != nil {
		return nil, err
	}

	switch append.targetInfo.Type {
	case targetBody:
		append.runFunc = appendBody
	default:
		return nil, fmt.Errorf("invalid target type: %s", append.targetInfo.Type)
	}

	return &append, nil
}

func newAppendPagination(cfg *common.Config) (transform, error) {
	append, err := newAppend(cfg)
	if err != nil {
		return nil, err
	}

	switch append.targetInfo.Type {
	case targetBody:
		append.runFunc = appendBody
	case targetHeader:
		append.runFunc = appendHeader
	case targetURLParams:
		append.runFunc = appendURLParams
	default:
		return nil, fmt.Errorf("invalid target type: %s", append.targetInfo.Type)
	}

	return &append, nil
}

func newAppend(cfg *common.Config) (appendt, error) {
	c := &appendConfig{}
	if err := cfg.Unpack(c); err != nil {
		return appendt{}, errors.Wrap(err, "fail to unpack the append configuration")
	}

	ti, err := getTargetInfo(c.Target)
	if err != nil {
		return appendt{}, err
	}

	return appendt{
		targetInfo:   ti,
		value:        c.Value,
		defaultValue: c.Default,
	}, nil
}

func (append *appendt) run(ctx transformContext, transformable *transformable) (*transformable, error) {
	value := append.value.Execute(ctx, transformable, append.defaultValue)
	if err := append.runFunc(ctx, transformable, append.targetInfo.Name, value); err != nil {
		return nil, err
	}
	return transformable, nil
}

func appendToCommonMap(m common.MapStr, key, val string) error {
	var value interface{} = val
	if found, _ := m.HasKey(key); found {
		prev, _ := m.GetValue(key)
		switch t := prev.(type) {
		case []string:
			value = append(t, val)
		case []interface{}:
			value = append(t, val)
		default:
			value = []interface{}{prev, val}
		}

	}
	if _, err := m.Put(key, value); err != nil {
		return err
	}
	return nil
}

func appendBody(ctx transformContext, transformable *transformable, key, value string) error {
	return appendToCommonMap(transformable.body, key, value)
}

func appendHeader(ctx transformContext, transformable *transformable, key, value string) error {
	transformable.header.Add(key, value)
	return nil
}

func appendURLParams(ctx transformContext, transformable *transformable, key, value string) error {
	transformable.url.Query().Add(key, value)
	return nil
}
