// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	beattest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestCtxAfterDoRequest(t *testing.T) {
	registerRequestTransforms()
	t.Cleanup(func() { registeredTransforms = newRegistry() })

	// mock timeNow func to return a fixed value
	timeNow = func() time.Time {
		t, _ := time.Parse(time.RFC3339, "2002-10-02T15:00:00Z")
		return t
	}
	t.Cleanup(func() { timeNow = time.Now })

	// test with dateCursorHandler to have different payloads each request
	testServer := httptest.NewServer(dateCursorHandler())
	t.Cleanup(testServer.Close)

	cfg := conf.MustNewConfigFrom(map[string]interface{}{
		"interval":       1,
		"request.method": "GET",
		"request.url":    testServer.URL,
		"request.transforms": []interface{}{
			map[string]interface{}{
				"set": map[string]interface{}{
					"target":  "url.params.$filter",
					"value":   "alertCreationTime ge [[.cursor.timestamp]]",
					"default": `alertCreationTime ge [[formatDate (now (parseDuration "-10m")) "2006-01-02T15:04:05Z"]]`,
				},
			},
		},
		"cursor": map[string]interface{}{
			"timestamp": map[string]interface{}{
				"value": `[[index .last_response.body "@timestamp"]]`,
			},
		},
	})

	config := defaultConfig()
	assert.NoError(t, cfg.Unpack(&config))

	log := logp.NewLogger("")
	ctx := context.Background()
	client, err := newHTTPClient(ctx, config, log)
	assert.NoError(t, err)

	requestFactory := newRequestFactory(config, log)
	pagination := newPagination(config, client, log)
	responseProcessor := newResponseProcessor(config, pagination, log)

	requester := newRequester(client, requestFactory, responseProcessor, log)

	trCtx := emptyTransformContext()
	trCtx.cursor = newCursor(config.Cursor, log)

	// first request
	assert.NoError(t, requester.doRequest(ctx, trCtx, statelessPublisher{&beattest.FakeClient{}}))

	assert.EqualValues(
		t,
		common.MapStr{"timestamp": "2002-10-02T15:00:00Z"},
		trCtx.cursorMap(),
	)
	assert.EqualValues(
		t,
		&common.MapStr{"@timestamp": "2002-10-02T15:00:00Z", "foo": "bar"},
		trCtx.firstEventClone(),
	)
	assert.EqualValues(
		t,
		&common.MapStr{"@timestamp": "2002-10-02T15:00:00Z", "foo": "bar"},
		trCtx.lastEventClone(),
	)
	lastResp := trCtx.lastResponseClone()
	// ignore since has dynamic date and content length values
	// and is not relevant
	lastResp.header = nil
	assert.EqualValues(t,
		&response{
			page: 1,
			url:  *(newURL(fmt.Sprintf("%s?%s", testServer.URL, "%24filter=alertCreationTime+ge+2002-10-02T14%3A50%3A00Z"))),
			body: common.MapStr{"@timestamp": "2002-10-02T15:00:00Z", "foo": "bar"},
		},
		lastResp,
	)

	// second request
	assert.NoError(t, requester.doRequest(ctx, trCtx, statelessPublisher{&beattest.FakeClient{}}))

	assert.EqualValues(
		t,
		common.MapStr{"timestamp": "2002-10-02T15:00:01Z"},
		trCtx.cursorMap(),
	)

	assert.EqualValues(
		t,
		&common.MapStr{"@timestamp": "2002-10-02T15:00:01Z", "foo": "bar"},
		trCtx.firstEventClone(),
	)

	assert.EqualValues(
		t,
		&common.MapStr{"@timestamp": "2002-10-02T15:00:01Z", "foo": "bar"},
		trCtx.lastEventClone(),
	)

	lastResp = trCtx.lastResponseClone()
	lastResp.header = nil
	assert.EqualValues(t,
		&response{
			page: 1,
			url:  *(newURL(fmt.Sprintf("%s?%s", testServer.URL, "%24filter=alertCreationTime+ge+2002-10-02T15%3A00%3A00Z"))),
			body: common.MapStr{"@timestamp": "2002-10-02T15:00:01Z", "foo": "bar"},
		},
		lastResp,
	)
}
