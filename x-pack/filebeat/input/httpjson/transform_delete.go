// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"fmt"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const deleteName = "delete"

type deleteConfig struct {
	Target string `config:"target"`
}

type delete struct {
	targetInfo targetInfo

	runFunc func(ctx *transformContext, transformable transformable, key string) error
}

func (delete) transformName() string { return deleteName }

func newDeleteRequest(cfg *conf.C, _ *logp.Logger) (transform, error) {
	delete, err := newDelete(cfg)
	if err != nil {
		return nil, err
	}

	switch delete.targetInfo.Type {
	case targetBody:
		delete.runFunc = deleteBody
	case targetHeader:
		delete.runFunc = deleteHeader
	case targetURLParams:
		delete.runFunc = deleteURLParams
	default:
		return nil, fmt.Errorf("invalid target type: %s", delete.targetInfo.Type)
	}

	return &delete, nil
}

func newDeleteResponse(cfg *conf.C, _ *logp.Logger) (transform, error) {
	delete, err := newDelete(cfg)
	if err != nil {
		return nil, err
	}

	switch delete.targetInfo.Type {
	case targetBody:
		delete.runFunc = deleteBody
	default:
		return nil, fmt.Errorf("invalid target type: %s", delete.targetInfo.Type)
	}

	return &delete, nil
}

func newDeletePagination(cfg *conf.C, _ *logp.Logger) (transform, error) {
	delete, err := newDelete(cfg)
	if err != nil {
		return nil, err
	}

	switch delete.targetInfo.Type {
	case targetBody:
		delete.runFunc = deleteBody
	case targetHeader:
		delete.runFunc = deleteHeader
	case targetURLParams:
		delete.runFunc = deleteURLParams
	default:
		return nil, fmt.Errorf("invalid target type: %s", delete.targetInfo.Type)
	}

	return &delete, nil
}

func newDelete(cfg *conf.C) (delete, error) {
	c := &deleteConfig{}
	if err := cfg.Unpack(c); err != nil {
		return delete{}, fmt.Errorf("fail to unpack the delete configuration: %w", err)
	}

	ti, err := getTargetInfo(c.Target)
	if err != nil {
		return delete{}, err
	}

	return delete{
		targetInfo: ti,
	}, nil
}

func (delete *delete) run(ctx *transformContext, tr transformable) (transformable, error) {
	if err := delete.runFunc(ctx, tr, delete.targetInfo.Name); err != nil {
		return transformable{}, err
	}
	return tr, nil
}

func deleteFromCommonMap(m mapstr.M, key string) error {
	if err := m.Delete(key); err != mapstr.ErrKeyNotFound { //nolint:errorlint // mapstr.ErrKeyNotFound is never wrapped by Delete.
		return err
	}
	return nil
}

func deleteBody(ctx *transformContext, transformable transformable, key string) error {
	return deleteFromCommonMap(transformable.body(), key)
}

func deleteHeader(ctx *transformContext, transformable transformable, key string) error {
	transformable.header().Del(key)
	return nil
}

func deleteURLParams(ctx *transformContext, transformable transformable, key string) error {
	url := transformable.url()
	q := url.Query()
	q.Del(key)
	url.RawQuery = q.Encode()
	transformable.setURL(url)
	return nil
}
