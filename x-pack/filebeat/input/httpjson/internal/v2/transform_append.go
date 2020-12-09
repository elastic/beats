// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const appendName = "append"

type appendConfig struct {
	Target  string    `config:"target"`
	Value   *valueTpl `config:"value"`
	Default *valueTpl `config:"default"`
}

type appendt struct {
	log          *logp.Logger
	targetInfo   targetInfo
	value        *valueTpl
	defaultValue *valueTpl

	runFunc func(ctx *transformContext, transformable transformable, key, val string) error
}

func (appendt) transformName() string { return appendName }

func newAppendRequest(cfg *common.Config, log *logp.Logger) (transform, error) {
	append, err := newAppend(cfg, log)
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

func newAppendResponse(cfg *common.Config, log *logp.Logger) (transform, error) {
	append, err := newAppend(cfg, log)
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

func newAppendPagination(cfg *common.Config, log *logp.Logger) (transform, error) {
	append, err := newAppend(cfg, log)
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

func newAppend(cfg *common.Config, log *logp.Logger) (appendt, error) {
	c := &appendConfig{}
	if err := cfg.Unpack(c); err != nil {
		return appendt{}, errors.Wrap(err, "fail to unpack the append configuration")
	}

	ti, err := getTargetInfo(c.Target)
	if err != nil {
		return appendt{}, err
	}

	return appendt{
		log:          log,
		targetInfo:   ti,
		value:        c.Value,
		defaultValue: c.Default,
	}, nil
}

func (append *appendt) run(ctx *transformContext, tr transformable) (transformable, error) {
	value := append.value.Execute(ctx, tr, append.defaultValue, append.log)
	if err := append.runFunc(ctx, tr, append.targetInfo.Name, value); err != nil {
		return transformable{}, err
	}
	return tr, nil
}

func appendToCommonMap(m common.MapStr, key, val string) error {
	if val == "" {
		return nil
	}
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

func appendBody(ctx *transformContext, transformable transformable, key, value string) error {
	return appendToCommonMap(transformable.body(), key, value)
}

func appendHeader(ctx *transformContext, transformable transformable, key, value string) error {
	if value == "" {
		return nil
	}
	transformable.header().Add(key, value)
	return nil
}

func appendURLParams(ctx *transformContext, transformable transformable, key, value string) error {
	if value == "" {
		return nil
	}
	url := transformable.url()
	q := url.Query()
	q.Add(key, value)
	url.RawQuery = q.Encode()
	transformable.setURL(url)
	return nil
}
