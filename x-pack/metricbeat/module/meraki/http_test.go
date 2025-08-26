// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package meraki

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp/logptest"
)

type requestConfig struct {
	startingAfter string
}

type T struct {
	thing string
}

func TestPaginatorGetAllPages(t *testing.T) {
	config := &requestConfig{}
	setStart := func(s string) { config.startingAfter = s }

	requestCount := 0
	doRequest := func() (*T, *resty.Response, error) {
		requestCount += 1
		headers := http.Header{}

		switch requestCount {
		case 1:
			assert.Equal(t, config.startingAfter, "")
			headers.Add("Link", "<https://api.meraki.com/api/v1/organizations/123456/appliance/uplink/statuses?startingAfter=0000-0000-0000>; rel=first, <https://api.meraki.com/api/v1/organizations/123456/appliance/uplink/statuses?startingAfter=1>; rel=next, <https://api.meraki.com/api/v1/organizations/123456/appliance/uplink/statuses?endingBefore=ZZZZ-ZZZZ-ZZZZ>; rel=last")
		case 2:
			assert.Equal(t, config.startingAfter, "1")
			headers.Add("Link", "<https://api.meraki.com/api/v1/organizations/123456/appliance/uplink/statuses?startingAfter=0000-0000-0000>; rel=first, <https://api.meraki.com/api/v1/organizations/123456/appliance/uplink/statuses?startingAfter=2>; rel=next, <https://api.meraki.com/api/v1/organizations/123456/appliance/uplink/statuses?endingBefore=ZZZZ-ZZZZ-ZZZZ>; rel=last")
		case 3:
			assert.Equal(t, config.startingAfter, "2")
		}

		return &T{thing: "val"}, &resty.Response{RawResponse: &http.Response{Header: headers}}, nil

	}

	var results []*T
	onSuccess := func(r *T) error {
		results = append(results, r)
		return nil
	}

	onError := func(_ error, _ *resty.Response) error {
		// not tested here
		return nil
	}

	err := NewPaginator(
		setStart,
		doRequest,
		onError,
		onSuccess,
		logptest.NewTestingLogger(t, ""),
	).GetAllPages()

	assert.NoError(t, err)
	assert.Equal(t, 3, requestCount)
	assert.Equal(t, 3, len(results))
}

func TestPaginatorGetAllPagesWithMaxPageLimit(t *testing.T) {
	setStart := func(_ string) {}

	requestCount := 0
	doRequest := func() (*T, *resty.Response, error) {
		requestCount += 1
		headers := http.Header{}
		// simlulates a broken API that always returns the same link header
		headers.Add("Link", "<https://api.meraki.com/api/v1/organizations/123456/appliance/uplink/statuses?startingAfter=0000-0000-0000>; rel=first, <https://api.meraki.com/api/v1/organizations/123456/appliance/uplink/statuses?startingAfter=1>; rel=next, <https://api.meraki.com/api/v1/organizations/123456/appliance/uplink/statuses?endingBefore=ZZZZ-ZZZZ-ZZZZ>; rel=last")
		return &T{thing: "val"}, &resty.Response{RawResponse: &http.Response{Header: headers}}, nil
	}

	onSuccess := func(r *T) error { return nil }
	onError := func(_ error, _ *resty.Response) error { return nil }

	err := NewPaginator(
		setStart,
		doRequest,
		onError,
		onSuccess,
		logptest.NewTestingLogger(t, ""),
	).GetAllPages()

	assert.NoError(t, err)
	assert.Equal(t, 100, requestCount)
}

func TestPaginatorGetAllPagesWithError(t *testing.T) {
	setStart := func(_ string) {}

	doRequest := func() (*T, *resty.Response, error) {
		return nil, &resty.Response{RawResponse: &http.Response{StatusCode: 500}}, fmt.Errorf("something went wrong")
	}

	onSuccess := func(_ *T) error {
		return nil
	}

	onError := func(err error, resp *resty.Response) error {
		assert.Error(t, err)
		assert.Equal(t, resp.StatusCode(), 500)
		return err
	}

	err := NewPaginator(
		setStart,
		doRequest,
		onError,
		onSuccess,
		logptest.NewTestingLogger(t, ""),
	).GetAllPages()

	assert.Error(t, err)
}

func TestPaginatorGetAllPagesWithMalformedLinkHeader(t *testing.T) {
	setStart := func(_ string) {}

	doRequest := func() (*T, *resty.Response, error) {
		headers := http.Header{}
		headers.Add("Link", "<http://foo.com/%zz>; rel=next")
		return nil, &resty.Response{RawResponse: &http.Response{Header: headers}}, nil
	}

	onSuccess := func(_ *T) error {
		return nil
	}

	onError := func(err error, _ *resty.Response) error {
		return err
	}

	err := NewPaginator(
		setStart,
		doRequest,
		onError,
		onSuccess,
		logptest.NewTestingLogger(t, ""),
	).GetAllPages()

	assert.Error(t, err)
}

func TestPaginatorGetAllPagesWithLinkHeaderURLWithoutStartingAfter(t *testing.T) {
	setStart := func(_ string) {}

	doRequest := func() (*T, *resty.Response, error) {
		headers := http.Header{}
		headers.Add("Link", "<https://api.meraki.com/api/v1/organizations/123456/appliance/uplink/statuses>; rel=next")
		return nil, &resty.Response{RawResponse: &http.Response{Header: headers}}, nil
	}

	onSuccess := func(_ *T) error { return nil }

	onError := func(err error, _ *resty.Response) error {
		return err
	}

	err := NewPaginator(
		setStart,
		doRequest,
		onError,
		onSuccess,
		logptest.NewTestingLogger(t, ""),
	).GetAllPages()

	assert.Error(t, err)
}

func TestPaginatorGetAllPagesWithMissingLinkHeader(t *testing.T) {
	setStart := func(_ string) {}

	doRequest := func() (*T, *resty.Response, error) {
		return &T{thing: "val"}, &resty.Response{RawResponse: &http.Response{Header: http.Header{}}}, nil
	}

	onSuccess := func(val *T) error {
		assert.Equal(t, val.thing, "val")
		return nil
	}

	onError := func(err error, _ *resty.Response) error {
		return err
	}

	err := NewPaginator(
		setStart,
		doRequest,
		onError,
		onSuccess,
		logptest.NewTestingLogger(t, ""),
	).GetAllPages()

	assert.NoError(t, err)
}

func TestPaginatorGetAllPagesWithErrorOnSuccess(t *testing.T) {
	setStart := func(_ string) {}

	doRequest := func() (*T, *resty.Response, error) {
		return nil, nil, nil
	}

	onSuccess := func(_ *T) error {
		return fmt.Errorf("something went wrong")
	}

	onError := func(err error, _ *resty.Response) error {
		return err
	}

	err := NewPaginator(
		setStart,
		doRequest,
		onError,
		onSuccess,
		logptest.NewTestingLogger(t, ""),
	).GetAllPages()

	assert.Error(t, err)
}
