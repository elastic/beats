// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/elastic/beats/v7/libbeat/logp"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var errNewURLValueNotSet = errors.New("the new url.value was not set")

const setName = "set"

type setConfig struct {
	Target              string    `config:"target"`
	Value               *valueTpl `config:"value"`
	Default             *valueTpl `config:"default"`
	FailOnTemplateError bool      `config:"fail_on_template_error"`
	ValueType           string    `config:"value_type"`
}

type set struct {
	log                 *logp.Logger
	targetInfo          targetInfo
	value               *valueTpl
	defaultValue        *valueTpl
	failOnTemplateError bool
	valueType           valueType

	runFunc func(ctx *transformContext, transformable transformable, key string, val interface{}) error
}

func (set) transformName() string { return setName }

func newSetRequestPagination(cfg *conf.C, log *logp.Logger) (transform, error) {
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

func newSetResponse(cfg *conf.C, log *logp.Logger) (transform, error) {
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

//nolint:dupl // Bad linter! Claims duplication with newAppend. The duplication exists but is not resolvable without parameterised types.
func newSet(cfg *conf.C, log *logp.Logger) (set, error) {
	c := &setConfig{}
	if err := cfg.Unpack(c); err != nil {
		return set{}, fmt.Errorf("fail to unpack the set configuration: %w", err)
	}

	ti, err := getTargetInfo(c.Target)
	if err != nil {
		return set{}, err
	}

	vt, err := newValueType(c.ValueType)
	if err != nil {
		return set{}, err
	}

	return set{
		log:                 log,
		targetInfo:          ti,
		value:               c.Value,
		defaultValue:        c.Default,
		failOnTemplateError: c.FailOnTemplateError,
		valueType:           vt,
	}, nil
}

func (set *set) run(ctx *transformContext, tr transformable) (transformable, error) {
	value, err := set.value.Execute(ctx, tr, set.defaultValue, set.log)
	if err != nil && set.failOnTemplateError {
		return transformable{}, err
	}
	if value == "" {
		return tr, nil
	}
	converted, err := set.valueType.convertToType(value)
	if err != nil {
		return transformable{}, fmt.Errorf("can't convert template value to %s: %w", set.valueType, err)
	}
	if err := set.runFunc(ctx, tr, set.targetInfo.Name, converted); err != nil {
		return transformable{}, err
	}
	return tr, nil
}

func setToCommonMap(m mapstr.M, key string, val interface{}) error {
	if _, err := m.Put(key, val); err != nil {
		return err
	}
	return nil
}

func setBody(ctx *transformContext, transformable transformable, key string, value interface{}) error {
	return setToCommonMap(transformable.body(), key, value)
}

func setHeader(ctx *transformContext, transformable transformable, key string, value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("headers can only contain string values, but got: %T", value)
	}
	transformable.header().Add(key, v)
	return nil
}

func setURLParams(ctx *transformContext, transformable transformable, key string, value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("URL params can only contain string values, but got: %T", value)
	}
	url := transformable.url()
	q := url.Query()
	q.Set(key, v)
	url.RawQuery = q.Encode()
	transformable.setURL(url)
	return nil
}

func setURLValue(ctx *transformContext, transformable transformable, _ string, value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("URL value can only contain string values, but got: %T", value)
	}
	url, err := url.Parse(v)
	if err != nil {
		return errNewURLValueNotSet
	}
	transformable.setURL(*url)
	return nil
}
