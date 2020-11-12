// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const logName = "httpjson.transforms"

type transformsConfig []*common.Config

type transforms []transform

type transformContext struct {
	cursor       *cursor
	lastEvent    *common.MapStr
	lastResponse *transformable
}

func emptyTransformContext() transformContext {
	return transformContext{
		cursor:       &cursor{},
		lastEvent:    &common.MapStr{},
		lastResponse: emptyTransformable(),
	}
}

type transformable struct {
	body   common.MapStr
	header http.Header
	url    url.URL
}

func (t *transformable) clone() *transformable {
	if t == nil {
		return emptyTransformable()
	}
	return &transformable{
		body: func() common.MapStr {
			if t.body == nil {
				return common.MapStr{}
			}
			return t.body.Clone()
		}(),
		header: func() http.Header {
			if t.header == nil {
				return http.Header{}
			}
			return t.header.Clone()
		}(),
		url: t.url,
	}
}

func emptyTransformable() *transformable {
	return &transformable{
		body:   common.MapStr{},
		header: http.Header{},
	}
}

type transform interface {
	transformName() string
}

type basicTransform interface {
	transform
	run(transformContext, *transformable) (*transformable, error)
}

type maybeMsg struct {
	err error
	msg common.MapStr
}

func (e maybeMsg) failed() bool { return e.err != nil }

func (e maybeMsg) Error() string { return e.err.Error() }

// newTransformsFromConfig creates a list of transforms from a list of free user configurations.
func newTransformsFromConfig(config transformsConfig, namespace string, log *logp.Logger) (transforms, error) {
	var trans transforms

	for _, tfConfig := range config {
		if len(tfConfig.GetFields()) != 1 {
			return nil, errors.Errorf(
				"each transform must have exactly one action, but found %d actions",
				len(tfConfig.GetFields()),
			)
		}

		actionName := tfConfig.GetFields()[0]
		cfg, err := tfConfig.Child(actionName, -1)
		if err != nil {
			return nil, err
		}

		constructor, found := registeredTransforms.get(namespace, actionName)
		if !found {
			return nil, errors.Errorf("the transform %s does not exist. Valid transforms: %s", actionName, registeredTransforms.String())
		}

		cfg.PrintDebugf("Configure transform '%v' with:", actionName)
		transform, err := constructor(cfg, log)
		if err != nil {
			return nil, err
		}

		trans = append(trans, transform)
	}

	return trans, nil
}

func newBasicTransformsFromConfig(config transformsConfig, namespace string, log *logp.Logger) ([]basicTransform, error) {
	ts, err := newTransformsFromConfig(config, namespace, log)
	if err != nil {
		return nil, err
	}

	var rts []basicTransform
	for _, t := range ts {
		rt, ok := t.(basicTransform)
		if !ok {
			return nil, fmt.Errorf("transform %s is not a valid %s transform", t.transformName(), namespace)
		}
		rts = append(rts, rt)
	}

	return rts, nil
}
