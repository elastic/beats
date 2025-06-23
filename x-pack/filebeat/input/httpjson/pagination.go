// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/mito/lib/xml"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const paginationNamespace = "pagination"

type pagination struct {
	client         *httpClient
	requestFactory *requestFactory
	decoder        decoderFunc
	status         status.StatusReporter
	log            *logp.Logger
}

func newPagination(config config, client *httpClient, stat status.StatusReporter, log *logp.Logger) *pagination {
	pagination := &pagination{client: client, status: stat, log: log}
	if config.Response == nil {
		return pagination
	}

	pagination.decoder = registeredDecoders[config.Response.DecodeAs]

	if len(config.Response.Pagination) == 0 {
		return pagination
	}

	rts, _ := newBasicTransformsFromConfig(registeredTransforms, config.Request.Transforms, requestNamespace, stat, log)
	pts, _ := newBasicTransformsFromConfig(registeredTransforms, config.Response.Pagination, paginationNamespace, stat, log)

	body := func() *mapstr.M {
		if config.Response.RequestBodyOnPagination {
			return config.Request.Body
		}
		return &mapstr.M{}
	}()

	requestFactory := newPaginationRequestFactory(
		config.Request.Method,
		config.Request.EncodeAs,
		*config.Request.URL.URL,
		body,
		append(rts, pts...),
		config.Auth,
		stat,
		log,
	)
	pagination.requestFactory = requestFactory
	return pagination
}

func newPaginationRequestFactory(method, encodeAs string, url url.URL, body *mapstr.M, ts []basicTransform, authConfig *authConfig, stat status.StatusReporter, log *logp.Logger) *requestFactory {
	// config validation already checked for errors here
	rf := &requestFactory{
		url:        url,
		method:     method,
		body:       body,
		transforms: ts,
		log:        log,
		encoder:    registeredEncoders[encodeAs],
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

	xmlDetails map[string]xml.Detail

	isFirst bool
	done    bool

	n int64

	status status.StatusReporter
}

func (p *pagination) newPageIterator(stdCtx context.Context, trCtx *transformContext, resp *http.Response, xmlDetails map[string]xml.Detail, stat status.StatusReporter) *pageIterator {
	return &pageIterator{
		pagination: p,
		stdCtx:     stdCtx,
		trCtx:      trCtx,
		resp:       resp,
		xmlDetails: xmlDetails,
		isFirst:    true,
		status:     stat,
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
	switch {
	case err == nil:
		// OK
	case errors.Is(err, errNewURLValueNotSet),
		errors.Is(err, errEmptyTemplateResult),
		errors.Is(err, errExecutingTemplate):
		// If this error happens here it means a transform
		// did not find any new value and we can stop paginating without error.
		iter.done = true
		return nil, false, nil
	default:
		return nil, false, err
	}

	//nolint:bodyclose // response body is closed through drainBody method
	resp, err := iter.pagination.client.do(iter.stdCtx, httpReq)
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
	bodyBytes, err := io.ReadAll(iter.resp.Body)
	if err != nil {
		iter.status.UpdateStatus(status.Degraded, "failed to read page body: "+err.Error())
		return nil, err
	}
	iter.resp.Body.Close()

	var r response
	r.header = iter.resp.Header
	r.url = *iter.resp.Request.URL

	// we set the page number before increasing its value
	// because the first page needs to be 0 for every interval
	r.page = iter.n
	iter.n++

	if len(bodyBytes) > 0 {
		if iter.pagination.decoder != nil {
			err = iter.pagination.decoder(bodyBytes, &r)
		} else {
			r.xmlDetails = iter.xmlDetails
			err = decode(iter.resp.Header.Get("Content-Type"), bodyBytes, &r)
		}
		if err != nil {
			iter.status.UpdateStatus(status.Degraded, "failed to decode page: "+err.Error())
			return nil, err
		}
	}

	return &r, nil
}
