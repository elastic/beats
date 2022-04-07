// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package o365audit

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Azure/go-autorest/autorest"

	"github.com/elastic/beats/v8/x-pack/filebeat/input/o365audit/poll"
)

// paginator is a decorator around a poll.Transaction to parse paginated requests.
type paginator struct {
	url   string
	inner poll.Transaction
}

// String returns the printable representation of this transaction.
func (p paginator) String() string {
	return fmt.Sprintf("pager for url:`%s` inner:%s", p.url, p.inner)
}

// RequestDecorators returns the decorators used to perform a request.
func (p paginator) RequestDecorators() []autorest.PrepareDecorator {
	return []autorest.PrepareDecorator{
		autorest.WithBaseURL(p.url),
	}
}

// OnResponse parses the response using the wrapped transaction.
func (p paginator) OnResponse(r *http.Response) []poll.Action {
	return p.inner.OnResponse(r)
}

// Delay returns the delay for the wrapped transaction.
func (p paginator) Delay() time.Duration {
	return p.inner.Delay()
}

func newPager(pageUrl string, inner poll.Transaction) poll.Transaction {
	return paginator{
		url:   pageUrl,
		inner: inner,
	}
}

// The documentation mentions NextPageUri, but shows NetPageUrl in the examples.
var nextPageHeaders = []string{
	"NextPageUri",
	"NextPageUrl",
}

func getNextPage(response *http.Response) (url string, found bool) {
	for _, h := range nextPageHeaders {
		if urls, found := response.Header[h]; found && len(urls) > 0 {
			return urls[0], true
		}
	}
	return "", false
}
