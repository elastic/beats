// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package meraki

import (
	"fmt"
	"net/url"

	"github.com/go-resty/resty/v2"
	"github.com/tomnomnom/linkheader"
)

type paginator[T any] struct {
	setStart  func(string)
	doRequest func() (T, *resty.Response, error)
	onError   func(error, *resty.Response) error
	onSuccess func(T) error
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
	}
}

func (p *paginator[T]) GetAllPages() error {
	hasMorePages := true

	for hasMorePages {
		val, res, err := p.doRequest()

		if err != nil {
			return p.onError(err, res)
		}

		if err := p.onSuccess(val); err != nil {
			return err
		}

		hasMorePages = false
		linkHeader := res.Header().Get("Link")
		for _, link := range linkheader.Parse(linkHeader) {
			if link.Rel == "next" {
				nextURL, err := url.Parse(link.URL)
				if err != nil {
					return fmt.Errorf("could not parse URL for next page in Link header: '%s'", linkHeader)
				}

				if start := nextURL.Query().Get("startingAfter"); start != "" {
					p.setStart(start)
					hasMorePages = true
					break
				}
			}
		}
	}

	return nil
}
