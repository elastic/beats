// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const logName = "httpjson.transforms"

type transformsConfig []*common.Config

type transforms []transform

type transformContext struct {
	lock         sync.RWMutex
	cursor       *cursor
	firstEvent   *common.MapStr
	lastEvent    *common.MapStr
	lastResponse *response
}

func emptyTransformContext() *transformContext {
	return &transformContext{
		cursor:       &cursor{},
		lastEvent:    &common.MapStr{},
		firstEvent:   &common.MapStr{},
		lastResponse: &response{},
	}
}

func (ctx *transformContext) cursorMap() common.MapStr {
	ctx.lock.RLock()
	defer ctx.lock.RUnlock()
	return ctx.cursor.clone()
}

func (ctx *transformContext) lastEventClone() *common.MapStr {
	ctx.lock.RLock()
	defer ctx.lock.RUnlock()
	clone := ctx.lastEvent.Clone()
	return &clone
}

func (ctx *transformContext) firstEventClone() *common.MapStr {
	ctx.lock.RLock()
	defer ctx.lock.RUnlock()
	clone := ctx.firstEvent.Clone()
	return &clone
}

func (ctx *transformContext) lastResponseClone() *response {
	ctx.lock.RLock()
	defer ctx.lock.RUnlock()
	return ctx.lastResponse.clone()
}

func (ctx *transformContext) updateCursor() {
	ctx.lock.Lock()
	defer ctx.lock.Unlock()

	// we do not want to pass the cursor data to itself
	newCtx := emptyTransformContext()
	newCtx.lastEvent = ctx.lastEvent
	newCtx.firstEvent = ctx.firstEvent
	newCtx.lastResponse = ctx.lastResponse

	ctx.cursor.update(newCtx)
}

func (ctx *transformContext) updateLastEvent(e common.MapStr) {
	ctx.lock.Lock()
	defer ctx.lock.Unlock()
	*ctx.lastEvent = e
}

func (ctx *transformContext) updateFirstEvent(e common.MapStr) {
	ctx.lock.Lock()
	defer ctx.lock.Unlock()
	*ctx.firstEvent = e
}

func (ctx *transformContext) updateLastResponse(r response) {
	ctx.lock.Lock()
	defer ctx.lock.Unlock()
	*ctx.lastResponse = r
}

type transformable common.MapStr

func (tr transformable) access() common.MapStr {
	return common.MapStr(tr)
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

func (tr transformable) setBody(v common.MapStr) {
	tr.Put("body", v)
}

func (tr transformable) body() common.MapStr {
	val, err := tr.GetValue("body")
	if err != nil {
		// if it does not exist, initialize it
		body := common.MapStr{}
		tr.setBody(body)
		return body
	}

	body, _ := val.(common.MapStr)

	return body
}

func (tr transformable) setURL(v url.URL) {
	tr.Put("url", v)
}

func (tr transformable) url() url.URL {
	val, err := tr.GetValue("url")
	if err != nil {
		return url.URL{}
	}

	u, ok := val.(url.URL)
	if !ok {
		return url.URL{}
	}

	return u
}

type transform interface {
	transformName() string
}

type basicTransform interface {
	transform
	run(*transformContext, transformable) (transformable, error)
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
