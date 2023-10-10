// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/elastic/beats/v7/libbeat/common"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type transformsConfig []*conf.C

type transforms []transform

type transformContext struct {
	cursor        *cursor
	parentTrCtx   *transformContext
	firstEvent    *mapstr.M
	lastEvent     *mapstr.M
	lastResponse  *response
	firstResponse *response
}

func emptyTransformContext() *transformContext {
	return &transformContext{
		cursor:        &cursor{},
		lastEvent:     &mapstr.M{},
		firstEvent:    &mapstr.M{},
		lastResponse:  &response{},
		firstResponse: &response{},
	}
}

func (ctx *transformContext) cursorMap() mapstr.M {
	return ctx.cursor.clone()
}

func (ctx *transformContext) lastEventClone() *mapstr.M {
	clone := ctx.lastEvent.Clone()
	return &clone
}

func (ctx *transformContext) firstEventClone() *mapstr.M {
	clone := ctx.firstEvent.Clone()
	return &clone
}

func (ctx *transformContext) firstResponseClone() *response {
	return ctx.firstResponse.clone()
}

func (ctx *transformContext) lastResponseClone() *response {
	return ctx.lastResponse.clone()
}

func (ctx *transformContext) updateCursor() {
	// we do not want to pass the cursor data to itself
	newCtx := emptyTransformContext()
	newCtx.lastEvent = ctx.lastEvent
	newCtx.firstEvent = ctx.firstEvent
	newCtx.lastResponse = ctx.lastResponse
	newCtx.firstResponse = ctx.firstResponse

	ctx.cursor.update(newCtx)
}

func (ctx *transformContext) clone() *transformContext {
	newCtx := emptyTransformContext()
	newCtx.lastEvent = ctx.lastEvent
	newCtx.firstEvent = ctx.firstEvent
	newCtx.lastResponse = ctx.lastResponse
	newCtx.firstResponse = ctx.firstResponse
	newCtx.cursor = ctx.cursor
	newCtx.parentTrCtx = ctx
	return newCtx
}

func (ctx *transformContext) updateLastEvent(e mapstr.M) {
	*ctx.lastEvent = e
}

func (ctx *transformContext) updateFirstEvent(e mapstr.M) {
	*ctx.firstEvent = e
}

func (ctx *transformContext) updateLastResponse(r response) {
	*ctx.lastResponse = r
}

func (ctx *transformContext) updateFirstResponse(r response) {
	*ctx.firstResponse = r
}

func (ctx *transformContext) clearIntervalData() {
	ctx.lastEvent = &mapstr.M{}
	ctx.firstEvent = &mapstr.M{}
	ctx.lastResponse = &response{}
}

type transformable mapstr.M

func (tr transformable) access() mapstr.M {
	return mapstr.M(tr)
}

func (tr transformable) Put(k string, v interface{}) {
	_, _ = tr.access().Put(k, v)
}

func (tr transformable) GetValue(k string) (interface{}, error) {
	return tr.access().GetValue(k)
}

func (tr transformable) Clone() transformable {
	return transformable(tr.access().Clone())
}

func (tr transformable) setHeader(v http.Header) {
	tr.Put("header", v)
}

func (tr transformable) header() http.Header {
	val, err := tr.GetValue("header")
	if err != nil {
		// if it does not exist, initialize it
		header := http.Header{}
		tr.setHeader(header)
		return header
	}

	header, _ := val.(http.Header)

	return header
}

func (tr transformable) setBody(v mapstr.M) {
	tr.Put("body", v)
}

func (tr transformable) body() mapstr.M {
	val, err := tr.GetValue("body")
	if err != nil {
		// if it does not exist, initialize it
		body := mapstr.M{}
		tr.setBody(body)
		return body
	}

	body, _ := val.(mapstr.M)

	return body
}

func (tr transformable) setURL(v url.URL) {
	tr.Put("url", &v)
}

func (tr transformable) url() url.URL {
	val, err := tr.GetValue("url")
	if err != nil {
		return url.URL{}
	}

	u, ok := val.(*url.URL)
	if !ok {
		return url.URL{}
	}

	return *u
}

type transform interface {
	transformName() string
}

type basicTransform interface {
	transform
	run(*transformContext, transformable) (transformable, error)
}

// newTransformsFromConfig creates a list of transforms from a list of free user configurations.
func newTransformsFromConfig(registeredTransforms registry, config transformsConfig, namespace string, log *logp.Logger) (transforms, error) {
	trans := make(transforms, 0, len(config))
	for _, tfConfig := range config {
		if len(tfConfig.GetFields()) != 1 {
			return nil, fmt.Errorf(
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
			return nil, fmt.Errorf("the transform %s does not exist. Valid transforms: %s", actionName, registeredTransforms)
		}

		common.PrintConfigDebugf(cfg, "Configure transform '%v' with:", actionName)
		transform, err := constructor(cfg, log)
		if err != nil {
			return nil, err
		}

		trans = append(trans, transform)
	}

	return trans, nil
}

func newBasicTransformsFromConfig(registeredTransforms registry, config transformsConfig, namespace string, log *logp.Logger) ([]basicTransform, error) {
	ts, err := newTransformsFromConfig(registeredTransforms, config, namespace, log)
	if err != nil {
		return nil, err
	}

	rts := make([]basicTransform, 0, len(ts))
	for _, t := range ts {
		rt, ok := t.(basicTransform)
		if !ok {
			return nil, fmt.Errorf("transform %s is not a valid %s transform", t.transformName(), namespace)
		}
		rts = append(rts, rt)
	}

	return rts, nil
}

type valueType string

const (
	valueTypeString valueType = "string"
	valueTypeJSON   valueType = "json"
	valueTypeInt    valueType = "int"
)

func newValueType(s string) (valueType, error) {
	vt := valueType(s)
	if vt == "" {
		return valueTypeString, nil
	}
	switch vt {
	case valueTypeString, valueTypeJSON, valueTypeInt:
		return vt, nil
	default:
		return "", fmt.Errorf("invalid value_type: %s", s)
	}
}

func (vt valueType) convertToType(v string) (interface{}, error) {
	switch vt {
	case valueTypeString:
		return v, nil
	case valueTypeInt:
		return strconv.ParseInt(v, 10, 64)
	case valueTypeJSON:
		var o interface{}
		if err := json.Unmarshal([]byte(v), &o); err != nil {
			return nil, err
		}
		return o, nil
	default:
		return nil, fmt.Errorf("can't convert to unknown value_type: %s", vt)
	}
}
