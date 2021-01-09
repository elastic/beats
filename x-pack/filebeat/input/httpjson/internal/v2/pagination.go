// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const paginationNamespace = "pagination"

func registerPaginationTransforms() {
	registerTransform(paginationNamespace, appendName, newAppendPagination)
	registerTransform(paginationNamespace, deleteName, newDeletePagination)
	registerTransform(paginationNamespace, setName, newSetPagination)
}

type pagination struct {
	log            *logp.Logger
	httpClient     *httpClient
	requestFactory *requestFactory
}

func newPagination(config config, httpClient *httpClient, log *logp.Logger) *pagination {
	pagination := &pagination{httpClient: httpClient, log: log}
	if config.Response == nil || len(config.Response.Pagination) == 0 {
		return pagination
	}

	rts, _ := newBasicTransformsFromConfig(config.Request.Transforms, requestNamespace, log)
	pts, _ := newBasicTransformsFromConfig(config.Response.Pagination, paginationNamespace, log)

	body := func() *common.MapStr {
		if config.Response.RequestBodyOnPagination {
			return config.Request.Body
		}
		return &common.MapStr{}
	}()

	requestFactory := newPaginationRequestFactory(
		config.Request.Method,
		*config.Request.URL.URL,
		body,
		append(rts, pts...),
		config.Auth,
		log,
	)
	pagination.requestFactory = requestFactory
	return pagination
}

func newPaginationRequestFactory(method string, url url.URL, body *common.MapStr, ts []basicTransform, authConfig *authConfig, log *logp.Logger) *requestFactory {
	// config validation already checked for errors here
	rf := &requestFactory{
		url:        url,
		method:     method,
		body:       body,
		transforms: ts,
		log:        log,
	}
	if authConfig != nil && authConfig.Basic.isEnabled() {
		rf.user = authConfig.Basic.User
		rf.password = authConfig.Basic.Password
	}
	return rf
}

type pageIterator struct {
	pagination *pagination

	stdCtx context.Context
	trCtx  *transformContext

	resp *http.Response

	isFirst bool
	done    bool

	n int
}

func (p *pagination) newPageIterator(stdCtx context.Context, trCtx *transformContext, resp *http.Response) *pageIterator {
	return &pageIterator{
		pagination: p,
		stdCtx:     stdCtx,
		trCtx:      trCtx,
		resp:       resp,
		isFirst:    true,
	}
}

func (iter *pageIterator) next() (*response, bool, error) {
	if iter == nil || iter.resp == nil || iter.done {
		return nil, false, nil
	}

	if iter.isFirst {
		iter.isFirst = false
		tr, err := iter.getPage()
		if err != nil {
			return nil, false, err
		}
		if iter.pagination.requestFactory == nil {
			iter.done = true
		}
		return tr, true, nil
	}

	httpReq, err := iter.pagination.requestFactory.newHTTPRequest(iter.stdCtx, iter.trCtx)
	if err != nil {
		if err == errNewURLValueNotSet {
			// if this error happens here it means the transform used to pick the new url.value
			// did not find any new value and we can stop paginating without error
			iter.done = true
			return nil, false, nil
		}
		return nil, false, err
	}

	resp, err := iter.pagination.httpClient.do(iter.stdCtx, iter.trCtx, httpReq)
	if err != nil {
		return nil, false, err
	}

	iter.resp = resp

	r, err := iter.getPage()
	if err != nil {
		return nil, false, err
	}

	if r.body == nil {
		iter.pagination.log.Debug("finished pagination because there is no body")
		iter.done = true
		return nil, false, nil
	}

	return r, true, nil
}

func (iter *pageIterator) getPage() (*response, error) {
	bodyBytes, err := ioutil.ReadAll(iter.resp.Body)
	if err != nil {
		return nil, err
	}
	iter.resp.Body.Close()
	iter.n += 1

	var r response
	r.header = iter.resp.Header
	r.url = *iter.resp.Request.URL
	r.page = iter.n

	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &r.body); err != nil {
			return nil, err
		}
	}

	return &r, nil
}
