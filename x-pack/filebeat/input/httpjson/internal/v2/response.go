// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"context"
	"net/http"
	"net/url"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const responseNamespace = "response"

func registerResponseTransforms() {
	registerTransform(responseNamespace, appendName, newAppendResponse)
	registerTransform(responseNamespace, deleteName, newDeleteResponse)
	registerTransform(responseNamespace, setName, newSetResponse)
}

type response struct {
	page   int
	url    url.URL
	header http.Header
	body   interface{}
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
	case common.MapStr:
		clone.body = t.Clone()
	case map[string]interface{}:
		clone.body = common.MapStr(t).Clone()
	}

	return clone
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

func (rp *responseProcessor) startProcessing(stdCtx context.Context, trCtx *transformContext, resp *http.Response) (<-chan maybeMsg, error) {
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

			if !hasNext {
				return
			}

			respTrs := page.asTransformables(rp.log)

			if len(respTrs) == 0 {
				return
			}

			trCtx.updateLastResponse(*page)

			rp.log.Debugf("last received page: %#v", trCtx.lastResponse)

			for _, tr := range respTrs {
				for _, t := range rp.transforms {
					tr, err = t.run(trCtx, tr)
					if err != nil {
						ch <- maybeMsg{err: err}
						return
					}
				}

				if rp.split == nil {
					ch <- maybeMsg{msg: tr.body()}
					rp.log.Debug("no split found: continuing")
					continue
				}

				if err := rp.split.run(trCtx, tr, ch); err != nil {
					switch err {
					case errEmptyField:
						// nothing else to send for this page
						rp.log.Debug("split operation finished")
						continue
					case errEmptyRootField:
						// root field not found, most likely the response is empty
						rp.log.Debug(err)
						return
					default:
						rp.log.Debug("split operation failed")
						ch <- maybeMsg{err: err}
						return
					}
				}
			}
		}
	}()

	return ch, nil
}

func (resp *response) asTransformables(log *logp.Logger) []transformable {
	var ts []transformable

	convertAndAppend := func(m map[string]interface{}) {
		tr := transformable{}
		tr.setHeader(resp.header.Clone())
		tr.setURL(resp.url)
		tr.setBody(common.MapStr(m).Clone())
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

func (resp *response) templateValues() common.MapStr {
	if resp == nil {
		return common.MapStr{}
	}
	return common.MapStr{
		"header":    resp.header.Clone(),
		"page":      resp.page,
		"url.value": resp.url.String(),
		"params":    resp.url.Query(),
		"body":      resp.body,
	}
}
