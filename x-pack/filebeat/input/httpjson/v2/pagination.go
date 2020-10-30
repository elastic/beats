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
	httpClient     *http.Client
	requestFactory *requestFactory
}

func newPagination(config config, httpClient *http.Client, log *logp.Logger) *pagination {
	pagination := &pagination{httpClient: httpClient}
	if config.Response == nil {
		return pagination
	}
	ts, _ := newBasicTransformsFromConfig(config.Response.Pagination, paginationNamespace)
	requestFactory := newPaginationRequestFactory(
		config.Request.Method,
		*config.Request.URL.URL,
		ts,
		config.Auth,
		log,
	)
	pagination.requestFactory = requestFactory
	return pagination
}

func newPaginationRequestFactory(method string, url url.URL, ts []basicTransform, authConfig *authConfig, log *logp.Logger) *requestFactory {
	// config validation already checked for errors here
	rf := &requestFactory{
		url:        url,
		method:     method,
		body:       &common.MapStr{},
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
	trCtx  transformContext

	resp *http.Response

	isFirst bool
}

func (p *pagination) newPageIterator(stdCtx context.Context, trCtx transformContext, resp *http.Response) *pageIterator {
	return &pageIterator{
		pagination: p,
		stdCtx:     stdCtx,
		trCtx:      trCtx,
		resp:       resp,
		isFirst:    true,
	}
}

func (iter *pageIterator) next() (*transformable, bool, error) {
	if iter == nil || iter.resp == nil {
		return nil, false, nil
	}

	if iter.isFirst {
		iter.isFirst = false
		tr, err := iter.getPage()
		if err != nil {
			return nil, false, err
		}
		return tr, true, nil
	}

	httpReq, err := iter.pagination.requestFactory.newHTTPRequest(iter.stdCtx, iter.trCtx)
	if err != nil {
		return nil, false, err
	}

	resp, err := iter.pagination.httpClient.Do(httpReq)
	if err != nil {
		return nil, false, err
	}

	iter.resp = resp

	tr, err := iter.getPage()
	if err != nil {
		return nil, false, err
	}

	if len(tr.body) == 0 {
		return nil, false, nil
	}

	return tr, true, nil
}

func (iter *pageIterator) getPage() (*transformable, error) {
	bodyBytes, err := ioutil.ReadAll(iter.resp.Body)
	if err != nil {
		return nil, err
	}
	iter.resp.Body.Close()

	tr := emptyTransformable()
	tr.header = iter.resp.Header
	tr.url = *iter.resp.Request.URL

	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &tr.body); err != nil {
			return nil, err
		}
	}

	iter.trCtx.lastResponse = &transformable{
		body:   tr.body.Clone(),
		header: tr.header.Clone(),
		url:    tr.url,
	}

	return tr, nil
}
