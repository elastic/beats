// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"context"
	"net/http"
	"net/url"

	"github.com/elastic/beats/v7/libbeat/common"
)

const responseNamespace = "response"

func registerResponseTransforms() {
	registerTransform(responseNamespace, appendName, newAppendResponse)
	registerTransform(responseNamespace, deleteName, newDeleteResponse)
	registerTransform(responseNamespace, setName, newSetResponse)
}

type response struct {
	body   common.MapStr
	header http.Header
	url    *url.URL
}

type responseProcessor struct {
	splitTransform splitTransform
	transforms     []responseTransform
	pagination     *pagination
}

func (rp *responseProcessor) startProcessing(stdCtx context.Context, trCtx transformContext, resp *http.Response) (<-chan maybeEvent, error) {
	ch := make(chan maybeEvent)

	go func() {
		defer close(ch)

		iter := rp.pagination.newPageIterator(stdCtx, trCtx, resp)
		for iter.hasNext() {
			var err error
			page := iter.page()

			for _, t := range rp.transforms {
				page, err = t.run(trCtx, page)
				if err != nil {
					ch <- maybeEvent{err: err}
					return
				}
			}

			if rp.splitTransform == nil {
				continue
			}

			if err := rp.splitTransform.run(trCtx, page, ch); err != nil {
				ch <- maybeEvent{err: err}
				return
			}
		}
	}()

	return ch, nil
}
