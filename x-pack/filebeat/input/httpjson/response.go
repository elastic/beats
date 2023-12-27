// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/elastic/mito/lib/xml"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const responseNamespace = "response"

type response struct {
	page       int64
	url        url.URL
	header     http.Header
	xmlDetails map[string]xml.Detail
	body       interface{}
}

func (resp *response) clone() *response {
	clone := &response{
		page:   resp.page,
		header: resp.header.Clone(),
		url:    resp.url,
	}

	switch t := resp.body.(type) {
	case []interface{}:
		c := make([]interface{}, len(t))
		copy(c, t)
		clone.body = c
	case mapstr.M:
		clone.body = t.Clone()
	case map[string]interface{}:
		clone.body = mapstr.M(t).Clone()
	}

	return clone
}

func (resp *response) asTransformables(log *logp.Logger) []transformable {
	var ts []transformable

	convertAndAppend := func(m map[string]interface{}) {
		tr := transformable{}
		tr.setHeader(resp.header.Clone())
		tr.setURL(resp.url)
		tr.setBody(mapstr.M(m).Clone())
		ts = append(ts, tr)
	}

	switch tresp := resp.body.(type) {
	case []interface{}:
		for _, v := range tresp {
			m, ok := v.(map[string]interface{})
			if !ok {
				log.Debugf("events must be JSON objects, but got %T: skipping", v)
				continue
			}
			convertAndAppend(m)
		}
	case map[string]interface{}:
		convertAndAppend(tresp)
	default:
		log.Debugf("response is not a valid JSON")
	}

	return ts
}

func (resp *response) templateValues() mapstr.M {
	if resp == nil {
		return mapstr.M{}
	}
	return mapstr.M{
		"header": resp.header.Clone(),
		"page":   resp.page,
		"url": mapstr.M{
			"value":  resp.url.String(),
			"params": resp.url.Query(),
		},
		"body": resp.body,
	}
}

type responseProcessor struct {
	metrics    *inputMetrics
	transforms []basicTransform
	split      *split
	pagination *pagination
	xmlDetails map[string]xml.Detail
	log        *logp.Logger
}

func newResponseProcessor(config config, pagination *pagination, xmlDetails map[string]xml.Detail, metrics *inputMetrics, log *logp.Logger) []*responseProcessor {
	rps := make([]*responseProcessor, 0, len(config.Chain)+1)

	rp := &responseProcessor{
		pagination: pagination,
		xmlDetails: xmlDetails,
		log:        log,
		metrics:    metrics,
	}
	if config.Response == nil {
		rps = append(rps, rp)
		return rps
	}
	ts, _ := newBasicTransformsFromConfig(registeredTransforms, config.Response.Transforms, responseNamespace, log)
	rp.transforms = ts

	split, _ := newSplitResponse(config.Response.Split, log)

	rp.split = split

	rps = append(rps, rp)
	for _, ch := range config.Chain {
		rp := &responseProcessor{
			pagination: pagination,
			xmlDetails: xmlDetails,
			log:        log,
			metrics:    metrics,
		}
		// chain calls responseProcessor object
		if ch.Step != nil && ch.Step.Response != nil {
			split, _ := newSplitResponse(ch.Step.Response.Split, log)
			rp.split = split
		} else if ch.While != nil && ch.While.Response != nil {
			split, _ := newSplitResponse(ch.While.Response.Split, log)
			rp.split = split
		}

		rps = append(rps, rp)
	}

	return rps
}

func newChainResponseProcessor(config chainConfig, client *httpClient, xmlDetails map[string]xml.Detail, metrics *inputMetrics, log *logp.Logger) *responseProcessor {
	pagination := &pagination{client: client, log: log}

	rp := &responseProcessor{
		pagination: pagination,
		xmlDetails: xmlDetails,
		log:        log,
		metrics:    metrics,
	}
	if config.Step != nil {
		if config.Step.Response == nil {
			return rp
		}

		ts, _ := newBasicTransformsFromConfig(registeredTransforms, config.Step.Response.Transforms, responseNamespace, log)
		rp.transforms = ts

		split, _ := newSplitResponse(config.Step.Response.Split, log)

		rp.split = split
	} else if config.While != nil {
		if config.While.Response == nil {
			return rp
		}

		ts, _ := newBasicTransformsFromConfig(registeredTransforms, config.While.Response.Transforms, responseNamespace, log)
		rp.transforms = ts

		split, _ := newSplitResponse(config.While.Response.Split, log)

		rp.split = split
	}

	return rp
}

// handler is an event stream processor.
type handler interface {
	handleEvent(ctx context.Context, event mapstr.M)
	handleError(error)
}

func (rp *responseProcessor) startProcessing(ctx context.Context, trCtx *transformContext, resps []*http.Response, paginate bool, h handler) {
	trCtx.clearIntervalData()

	var npages int64
	for i, httpResp := range resps {
		iter := rp.pagination.newPageIterator(ctx, trCtx, httpResp, rp.xmlDetails)
		for {
			pageStartTime := time.Now()
			page, hasNext, err := iter.next()
			if err != nil {
				h.handleError(err)
				return
			}

			if !hasNext {
				if i+1 != len(resps) {
					break
				}
				return
			}

			respTrs := page.asTransformables(rp.log)

			if len(respTrs) == 0 {
				return
			}

			// last_response context object is updated here organically
			trCtx.updateLastResponse(*page)
			npages = page.page

			rp.log.Debugf("last received page: %#v", trCtx.lastResponse)

			for _, tr := range respTrs {
				for _, t := range rp.transforms {
					tr, err = t.run(trCtx, tr)
					if err != nil {
						h.handleError(err)
						return
					}
				}

				if rp.split == nil {
					h.handleEvent(ctx, tr.body())
					rp.log.Debug("no split found: continuing")
					continue
				}

				if err := rp.split.run(ctx, trCtx, tr, h); err != nil {
					switch err { //nolint:errorlint // run never returns a wrapped error.
					case errEmptyField:
						// nothing else to send for this page
						rp.log.Debug("split operation finished")
					case errEmptyRootField:
						// root field not found, most likely the response is empty
						rp.log.Debug(err)
					default:
						rp.log.Debug("split operation failed")
						h.handleError(err)
						return
					}
				}
			}

			rp.metrics.updatePageExecutionTime(pageStartTime)

			if !paginate {
				break
			}
		}
	}
	rp.metrics.updatePagesPerInterval(npages)
}
