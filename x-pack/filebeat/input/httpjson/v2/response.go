// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"context"
	"fmt"
	"net/http"
)

const responseNamespace = "response"

func registerResponseTransforms() {
	registerTransform(responseNamespace, appendName, newAppendResponse)
	registerTransform(responseNamespace, deleteName, newDeleteResponse)
	registerTransform(responseNamespace, setName, newSetResponse)
}

type responseProcessor struct {
	transforms []basicTransform
	split      *split
	pagination *pagination
}

func newResponseProcessor(config *responseConfig, pagination *pagination) *responseProcessor {
	rp := &responseProcessor{
		pagination: pagination,
	}
	if config == nil {
		return rp
	}
	ts, _ := newBasicTransformsFromConfig(config.Transforms, responseNamespace)
	rp.transforms = ts

	split, _ := newSplitResponse(config.Split)

	rp.split = split

	return rp
}

func (rp *responseProcessor) startProcessing(stdCtx context.Context, trCtx transformContext, resp *http.Response) (<-chan maybeEvent, error) {
	ch := make(chan maybeEvent)

	go func() {
		defer close(ch)

		iter := rp.pagination.newPageIterator(stdCtx, trCtx, resp)
		for {
			page, hasNext, err := iter.next()
			if err != nil {
				ch <- maybeEvent{err: err}
				return
			}

			if !hasNext || len(page.body) == 0 {
				return
			}

			for _, t := range rp.transforms {
				page, err = t.run(trCtx, page)
				if err != nil {
					fmt.Println("=== 2")
					ch <- maybeEvent{err: err}
					return
				}
			}

			if rp.split == nil {
				event, err := makeEvent(page.body)
				if err != nil {
					ch <- maybeEvent{err: err}
					return
				}
				ch <- maybeEvent{event: event}
				continue
			}

			if err := rp.split.run(trCtx, page, ch); err != nil {
				if err == errEmtpyField {
					// nothing else to send
					return
				}

				ch <- maybeEvent{err: err}
				return
			}
		}
	}()

	return ch, nil
}
