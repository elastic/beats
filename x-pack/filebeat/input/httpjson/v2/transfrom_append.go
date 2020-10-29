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

var (
	_ requestTransform    = &appendRequest{}
	_ responseTransform   = &appendResponse{}
	_ paginationTransform = &appendPagination{}
)

type appendConfig struct {
	Target  string    `config:"target"`
	Value   *valueTpl `config:"value"`
	Default string    `config:"default"`
}

type appendt struct {
	targetInfo   targetInfo
	value        *valueTpl
	defaultValue string

	run func(ctx transformContext, transformable *transformable, key, val string) error
}

func (appendt) transformName() string { return appendName }

type appendRequest struct {
	appendt
}

type appendResponse struct {
	appendt
}

type appendPagination struct {
	appendt
}

func newAppendRequest(cfg *common.Config) (transform, error) {
	append, err := newAppend(cfg)
	if err != nil {
		return nil, err
	}

	switch append.targetInfo.Type {
	case targetBody:
		append.run = appendBody
	case targetHeader:
		append.run = appendHeader
	case targetURLParams:
		append.run = appendURLParams
	default:
		return nil, fmt.Errorf("invalid target type: %s", append.targetInfo.Type)
	}

	return &appendRequest{appendt: append}, nil
}

func (appendReq *appendRequest) run(ctx transformContext, req *request) (*request, error) {
	transformable := &transformable{
		body:   req.body,
		header: req.header,
		url:    req.url,
	}
	if err := appendReq.appendt.runAppend(ctx, transformable); err != nil {
		return nil, err
	}
	return req, nil
}

func newAppendResponse(cfg *common.Config) (transform, error) {
	append, err := newAppend(cfg)
	if err != nil {
		return nil, err
	}

	switch append.targetInfo.Type {
	case targetBody:
		append.run = appendBody
	default:
		return nil, fmt.Errorf("invalid target type: %s", append.targetInfo.Type)
	}

	return &appendResponse{appendt: append}, nil
}

func (appendRes *appendResponse) run(ctx transformContext, res *response) (*response, error) {
	transformable := &transformable{
		body:   res.body,
		header: res.header,
		url:    res.url,
	}
	if err := appendRes.appendt.runAppend(ctx, transformable); err != nil {
		return nil, err
	}
	return res, nil
}

func newAppendPagination(cfg *common.Config) (transform, error) {
	append, err := newAppend(cfg)
	if err != nil {
		return nil, err
	}

	switch append.targetInfo.Type {
	case targetBody:
		append.run = appendBody
	case targetHeader:
		append.run = appendHeader
	case targetURLParams:
		append.run = appendURLParams
	default:
		return nil, fmt.Errorf("invalid target type: %s", append.targetInfo.Type)
	}

	return &appendPagination{appendt: append}, nil
}

func (appendPag *appendPagination) run(ctx transformContext, pag *pagination) (*pagination, error) {
	transformable := &transformable{
		body:   pag.body,
		header: pag.header,
		url:    pag.url,
	}
	if err := appendPag.appendt.runAppend(ctx, transformable); err != nil {
		return nil, err
	}
	return pag, nil
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

func (append *appendt) runAppend(ctx transformContext, transformable *transformable) error {
	value := append.value.Execute(ctx, transformable, append.defaultValue)
	return append.run(ctx, transformable, append.targetInfo.Name, value)
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
