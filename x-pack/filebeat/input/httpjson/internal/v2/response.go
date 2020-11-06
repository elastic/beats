// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"context"
	"net/http"

	"github.com/elastic/beats/v7/libbeat/logp"
)

const responseNamespace = "response"

func registerResponseTransforms() {
	registerTransform(responseNamespace, appendName, newAppendResponse)
	registerTransform(responseNamespace, deleteName, newDeleteResponse)
	registerTransform(responseNamespace, setName, newSetResponse)
}

type responseProcessor struct {
	log        *logp.Logger
	transforms []basicTransform
	split      *split
	pagination *pagination
}

func newResponseProcessor(config *responseConfig, pagination *pagination, log *logp.Logger) *responseProcessor {
	rp := &responseProcessor{
		pagination: pagination,
		log:        log,
	}
	if config == nil {
		return rp
	}
	ts, _ := newBasicTransformsFromConfig(config.Transforms, responseNamespace, log)
	rp.transforms = ts

	split, _ := newSplitResponse(config.Split, log)

	rp.split = split

	return rp
}

func (rp *responseProcessor) startProcessing(stdCtx context.Context, trCtx transformContext, resp *http.Response) (<-chan maybeMsg, error) {
	ch := make(chan maybeMsg)

	go func() {
		defer close(ch)

		iter := rp.pagination.newPageIterator(stdCtx, trCtx, resp)
		for {
			page, hasNext, err := iter.next()
			if err != nil {
				ch <- maybeMsg{err: err}
				return
			}

			if !hasNext || len(page.body) == 0 {
				return
			}

			*trCtx.lastResponse = *page.clone()

			rp.log.Debugf("last received page: %v", trCtx.lastResponse)

			for _, t := range rp.transforms {
				page, err = t.run(trCtx, page)
				if err != nil {
					ch <- maybeMsg{err: err}
					return
				}
			}

			if rp.split == nil {
				ch <- maybeMsg{msg: page.body}
				continue
			}

			if err := rp.split.run(trCtx, page, ch); err != nil {
				if err == errEmtpyField {
					// nothing else to send
					return
				}

				ch <- maybeMsg{err: err}
				return
			}
		}
	}()

	return ch, nil
}
