// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package meraki

import (
	"fmt"
	"net/url"

	"github.com/elastic/elastic-agent-libs/logp"

	"github.com/go-resty/resty/v2"
	"github.com/tomnomnom/linkheader"
)

const MAX_PAGES = 100

type paginator[T any] struct {
	setStart  func(string)
	doRequest func() (T, *resty.Response, error)
	onError   func(error, *resty.Response) error
	onSuccess func(T) error
	logger    *logp.Logger
}

func NewPaginator[T any](
	setStart func(string),
	doRequest func() (T, *resty.Response, error),
	onError func(error, *resty.Response) error,
	onSuccess func(T) error,
) *paginator[T] {
	return &paginator[T]{
		setStart:  setStart,
		doRequest: doRequest,
		onError:   onError,
		onSuccess: onSuccess,
		logger:    logp.NewLogger("meraki.paginator"),
	}
}

func (p *paginator[T]) GetAllPages() error {
	count := 0
	hasMorePages := true

	for hasMorePages {
		val, res, err := p.doRequest()

		if err != nil {
			p.logger.Debugf("onError; err: %w, res: %s", err, res)
			return p.onError(err, res)
		}

		if err := p.onSuccess(val); err != nil {
			return err
		}

		count += 1
		if count >= MAX_PAGES {
			p.logger.Errorf("maximum number of pages reached (%d); stopping", MAX_PAGES)
			return nil
		}

		hasMorePages = false
		linkHeader := res.Header().Get("Link")
		p.logger.Debugf("link header: %s", linkHeader)

		for _, link := range linkheader.Parse(linkHeader) {
			if link.Rel == "next" {
				nextURL, err := url.Parse(link.URL)
				if err != nil {
					return fmt.Errorf("could not parse URL for next page in Link header: '%s'", linkHeader)
				}

				if start := nextURL.Query().Get("startingAfter"); start != "" {
					p.logger.Debugf("parsed startingAfter from 'next' URL: %s", start)
					p.setStart(start)
					hasMorePages = true
					break
				} else {
					return fmt.Errorf("next URL in Link header does not have 'startingAfter' param: %s", nextURL.String())
				}
			}
		}
	}

	return nil
}
