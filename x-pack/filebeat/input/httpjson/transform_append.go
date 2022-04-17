// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"fmt"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"
)

const appendName = "append"

type appendConfig struct {
	Target              string    `config:"target"`
	Value               *valueTpl `config:"value"`
	Default             *valueTpl `config:"default"`
	FailOnTemplateError bool      `config:"fail_on_template_error"`
	ValueType           string    `config:"value_type"`
}

type appendt struct {
	log                 *logp.Logger
	targetInfo          targetInfo
	value               *valueTpl
	defaultValue        *valueTpl
	failOnTemplateError bool
	valueType           valueType

	runFunc func(ctx *transformContext, transformable transformable, key string, val interface{}) error
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

//nolint:dupl // Bad linter! Claims duplication with newSet. The duplication exists but is not resolvable without parameterised types.
func newAppend(cfg *common.Config, log *logp.Logger) (appendt, error) {
	c := &appendConfig{}
	if err := cfg.Unpack(c); err != nil {
		return appendt{}, fmt.Errorf("fail to unpack the append configuration: %w", err)
	}

	ti, err := getTargetInfo(c.Target)
	if err != nil {
		return appendt{}, err
	}

	vt, err := newValueType(c.ValueType)
	if err != nil {
		return appendt{}, err
	}

	return appendt{
		log:                 log,
		targetInfo:          ti,
		value:               c.Value,
		defaultValue:        c.Default,
		failOnTemplateError: c.FailOnTemplateError,
		valueType:           vt,
	}, nil
}

func (append *appendt) run(ctx *transformContext, tr transformable) (transformable, error) {
	value, err := append.value.Execute(ctx, tr, append.defaultValue, append.log)
	if err != nil && append.failOnTemplateError {
		return transformable{}, err
	}
	if value == "" {
		return tr, nil
	}
	converted, err := append.valueType.convertToType(value)
	if err != nil {
		return transformable{}, fmt.Errorf("can't convert template value to %s: %w", append.valueType, err)
	}
	if err := append.runFunc(ctx, tr, append.targetInfo.Name, converted); err != nil {
		return transformable{}, err
	}
	return tr, nil
}

func appendToCommonMap(m common.MapStr, key string, val interface{}) error {
	var value interface{}
	strVal, isString := val.(string)
	if found, _ := m.HasKey(key); found {
		prev, _ := m.GetValue(key)
		switch t := prev.(type) {
		case []string:
			if !isString {
				return fmt.Errorf("can't append a %T value to a string list", val)
			}
			value = append(t, strVal)
		case []interface{}:
			value = append(t, val)
		default:
			value = []interface{}{prev, val}
		}

	} else {
		value = []interface{}{val}
	}
	if _, err := m.Put(key, value); err != nil {
		return err
	}
	return nil
}

func appendBody(ctx *transformContext, transformable transformable, key string, value interface{}) error {
	return appendToCommonMap(transformable.body(), key, value)
}

func appendHeader(ctx *transformContext, transformable transformable, key string, value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("headers can only contain string values, but got: %T", value)
	}
	transformable.header().Add(key, v)
	return nil
}

func appendURLParams(ctx *transformContext, transformable transformable, key string, value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("URL params can only contain string values, but got: %T", value)
	}
	url := transformable.url()
	q := url.Query()
	q.Add(key, v)
	url.RawQuery = q.Encode()
	transformable.setURL(url)
	return nil
}
