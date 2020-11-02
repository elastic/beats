// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
)

const logName = "httpjson.transforms"

type transformsConfig []*common.Config

type transforms []transform

type transformContext struct {
	cursor       common.MapStr
	lastEvent    common.MapStr
	lastResponse *transformable
}

func emptyTransformContext() transformContext {
	return transformContext{
		cursor:       make(common.MapStr),
		lastEvent:    make(common.MapStr),
		lastResponse: emptyTransformable(),
	}
}

type transformable struct {
	body   common.MapStr
	header http.Header
	url    url.URL
}

func (t *transformable) clone() *transformable {
	return &transformable{
		body:   t.body.Clone(),
		header: t.header.Clone(),
		url:    t.url,
	}
}

func emptyTransformable() *transformable {
	return &transformable{
		body:   make(common.MapStr),
		header: make(http.Header),
	}
}

type transform interface {
	transformName() string
}

type basicTransform interface {
	transform
	run(transformContext, *transformable) (*transformable, error)
}

type maybeEvent struct {
	err   error
	event beat.Event
}

func (e maybeEvent) failed() bool { return e.err != nil }

func (e maybeEvent) Error() string { return e.err.Error() }

// newTransformsFromConfig creates a list of transforms from a list of free user configurations.
func newTransformsFromConfig(config transformsConfig, namespace string) (transforms, error) {
	var trans transforms

	for _, tfConfig := range config {
		if len(tfConfig.GetFields()) != 1 {
			return nil, errors.Errorf(
				"each transform must have exactly one action, but found %d actions (%v)",
				len(tfConfig.GetFields()),
				strings.Join(tfConfig.GetFields(), ","),
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
		transform, err := constructor(cfg)
		if err != nil {
			return nil, err
		}

		trans = append(trans, transform)
	}

	return trans, nil
}

func newBasicTransformsFromConfig(config transformsConfig, namespace string) ([]basicTransform, error) {
	ts, err := newTransformsFromConfig(config, namespace)
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

func (trans transforms) String() string {
	var s []string
	for _, p := range trans {
		s = append(s, p.transformName())
	}
	return strings.Join(s, ", ")
}
