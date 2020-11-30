// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"fmt"
	"net/url"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

var errNewURLValueNotSet = errors.New("the new url.value was not set")

const setName = "set"

type setConfig struct {
	Target  string    `config:"target"`
	Value   *valueTpl `config:"value"`
	Default *valueTpl `config:"default"`
}

type set struct {
	log          *logp.Logger
	targetInfo   targetInfo
	value        *valueTpl
	defaultValue *valueTpl

	runFunc func(ctx *transformContext, transformable transformable, key, val string) error
}

func (set) transformName() string { return setName }

func newSetRequest(cfg *common.Config, log *logp.Logger) (transform, error) {
	set, err := newSet(cfg, log)
	if err != nil {
		return nil, err
	}

	switch set.targetInfo.Type {
	case targetBody:
		set.runFunc = setBody
	case targetHeader:
		set.runFunc = setHeader
	case targetURLParams:
		set.runFunc = setURLParams
	default:
		return nil, fmt.Errorf("invalid target type: %s", set.targetInfo.Type)
	}

	return &set, nil
}

func newSetResponse(cfg *common.Config, log *logp.Logger) (transform, error) {
	set, err := newSet(cfg, log)
	if err != nil {
		return nil, err
	}

	switch set.targetInfo.Type {
	case targetBody:
		set.runFunc = setBody
	default:
		return nil, fmt.Errorf("invalid target type: %s", set.targetInfo.Type)
	}

	return &set, nil
}

func newSetPagination(cfg *common.Config, log *logp.Logger) (transform, error) {
	set, err := newSet(cfg, log)
	if err != nil {
		return nil, err
	}

	switch set.targetInfo.Type {
	case targetBody:
		set.runFunc = setBody
	case targetHeader:
		set.runFunc = setHeader
	case targetURLParams:
		set.runFunc = setURLParams
	case targetURLValue:
		set.runFunc = setURLValue
	default:
		return nil, fmt.Errorf("invalid target type: %s", set.targetInfo.Type)
	}

	return &set, nil
}

func newSet(cfg *common.Config, log *logp.Logger) (set, error) {
	c := &setConfig{}
	if err := cfg.Unpack(c); err != nil {
		return set{}, errors.Wrap(err, "fail to unpack the set configuration")
	}

	ti, err := getTargetInfo(c.Target)
	if err != nil {
		return set{}, err
	}

	return set{
		log:          log,
		targetInfo:   ti,
		value:        c.Value,
		defaultValue: c.Default,
	}, nil
}

func (set *set) run(ctx *transformContext, tr transformable) (transformable, error) {
	value := set.value.Execute(ctx, tr, set.defaultValue, set.log)
	if err := set.runFunc(ctx, tr, set.targetInfo.Name, value); err != nil {
		return transformable{}, err
	}
	return tr, nil
}

func setToCommonMap(m common.MapStr, key, val string) error {
	if val == "" {
		return nil
	}
	if _, err := m.Put(key, val); err != nil {
		return err
	}
	return nil
}

func setBody(ctx *transformContext, transformable transformable, key, value string) error {
	return setToCommonMap(transformable.body(), key, value)
}

func setHeader(ctx *transformContext, transformable transformable, key, value string) error {
	if value == "" {
		return nil
	}
	transformable.header().Add(key, value)
	return nil
}

func setURLParams(ctx *transformContext, transformable transformable, key, value string) error {
	if value == "" {
		return nil
	}
	url := transformable.url()
	q := url.Query()
	q.Set(key, value)
	url.RawQuery = q.Encode()
	transformable.setURL(url)
	return nil
}

func setURLValue(ctx *transformContext, transformable transformable, _, value string) error {
	// if the template processing did not find any value
	// we fail without parsing
	if value == "<no value>" || value == "" {
		return errNewURLValueNotSet
	}
	url, err := url.Parse(value)
	if err != nil {
		return errNewURLValueNotSet
	}
	transformable.setURL(*url)
	return nil
}
