// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	beattest "github.com/elastic/beats/v7/libbeat/publisher/testing"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestCtxAfterDoRequest(t *testing.T) {
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
	client, err := newHTTPClient(ctx, config, log, nil)
	assert.NoError(t, err)

	requestFactory, err := newRequestFactory(ctx, config, log, nil, nil)
	assert.NoError(t, err)
	pagination := newPagination(config, client, log)
	responseProcessor := newResponseProcessor(config, pagination, nil, nil, log)

	requester := newRequester(client, requestFactory, responseProcessor, log)

	trCtx := emptyTransformContext()
	trCtx.cursor = newCursor(config.Cursor, log)

	// first request
	assert.NoError(t, requester.doRequest(ctx, trCtx, statelessPublisher{&beattest.FakeClient{}}))

	assert.EqualValues(
		t,
		mapstr.M{"timestamp": "2002-10-02T15:00:00Z"},
		trCtx.cursorMap(),
	)
	assert.EqualValues(
		t,
		&mapstr.M{"@timestamp": "2002-10-02T15:00:00Z", "foo": "bar"},
		trCtx.firstEventClone(),
	)
	assert.EqualValues(
		t,
		&mapstr.M{"@timestamp": "2002-10-02T15:00:00Z", "foo": "bar"},
		trCtx.lastEventClone(),
	)
	lastResp := trCtx.lastResponseClone()
	// ignore since has dynamic date and content length values
	// and is not relevant
	lastResp.header = nil
	assert.EqualValues(t,
		&response{
			page: 0,
			url:  *(newURL(fmt.Sprintf("%s?%s", testServer.URL, "%24filter=alertCreationTime+ge+2002-10-02T14%3A50%3A00Z"))),
			body: mapstr.M{"@timestamp": "2002-10-02T15:00:00Z", "foo": "bar"},
		},
		lastResp,
	)

	// second request
	assert.NoError(t, requester.doRequest(ctx, trCtx, statelessPublisher{&beattest.FakeClient{}}))

	assert.EqualValues(
		t,
		mapstr.M{"timestamp": "2002-10-02T15:00:01Z"},
		trCtx.cursorMap(),
	)

	assert.EqualValues(
		t,
		&mapstr.M{"@timestamp": "2002-10-02T15:00:01Z", "foo": "bar"},
		trCtx.firstEventClone(),
	)

	assert.EqualValues(
		t,
		&mapstr.M{"@timestamp": "2002-10-02T15:00:01Z", "foo": "bar"},
		trCtx.lastEventClone(),
	)

	lastResp = trCtx.lastResponseClone()
	lastResp.header = nil
	assert.EqualValues(t,
		&response{
			page: 0,
			url:  *(newURL(fmt.Sprintf("%s?%s", testServer.URL, "%24filter=alertCreationTime+ge+2002-10-02T15%3A00%3A00Z"))),
			body: mapstr.M{"@timestamp": "2002-10-02T15:00:01Z", "foo": "bar"},
		},
		lastResp,
	)
}

func Test_newRequestFactory_UsesBasicAuthInChainedRequests(t *testing.T) {
	ctx := context.Background()
	log := logp.NewLogger("")
	cfg := defaultChainConfig()

	url, _ := url.Parse("https://example.com")
	cfg.Request.URL = &urlConfig{
		URL: url,
	}

	enabled := true
	user := "basicuser"
	password := "basicuser"
	cfg.Auth = &authConfig{
		Basic: &basicAuthConfig{
			Enabled:  &enabled,
			User:     user,
			Password: password,
		},
	}

	step := cfg.Chain[0].Step
	step.Auth = cfg.Auth

	while := cfg.Chain[0].While
	while.Auth = cfg.Auth

	type args struct {
		cfg   config
		step  *stepConfig
		while *whileConfig
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Step",
			args: args{
				cfg:   cfg,
				step:  step,
				while: nil,
			},
		},
		{
			name: "While",
			args: args{
				cfg:   cfg,
				step:  nil,
				while: while,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tt.args.cfg.Chain[0].Step = tt.args.step
			tt.args.cfg.Chain[0].While = tt.args.while
			requestFactories, err := newRequestFactory(ctx, tt.args.cfg, log, nil, nil)
			assert.NoError(t, err)
			assert.NotNil(t, requestFactories)
			for _, rf := range requestFactories {
				assert.Equal(t, rf.user, user)
				assert.Equal(t, rf.password, password)
			}

		})
	}
}

func Test_newChainHTTPClient(t *testing.T) {
	cfg := defaultChainConfig()
	cfg.Request.URL = &urlConfig{URL: &url.URL{}}
	ctx := context.Background()
	log := logp.NewLogger("newChainClientTestLogger")

	type args struct {
		ctx        context.Context
		authCfg    *authConfig
		requestCfg *requestConfig
		log        *logp.Logger
		p          []*Policy
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "newChainClientTest",
			args: args{
				ctx:        ctx,
				authCfg:    cfg.Auth,
				requestCfg: cfg.Request,
				log:        log,
				p:          nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newChainHTTPClient(tt.args.ctx, tt.args.authCfg, tt.args.requestCfg, tt.args.log, nil, tt.args.p...)
			assert.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

func Test_evaluateResponse(t *testing.T) {
	log := logp.NewLogger("newEvaluateResponseTestLogger")
	responseTrue := bytes.NewBufferString(`{"status": "completed"}`).Bytes()
	responseFalse := bytes.NewBufferString(`{"status": "initiated"}`).Bytes()

	type args struct {
		expression string
		data       []byte
		log        *logp.Logger
	}
	tests := []struct {
		name          string
		args          args
		expectedError string
		want          bool
	}{
		{
			name: "newEvaluateResponse_resultIsTrue",
			args: args{
				expression: `[[ eq .last_response.body.status "completed" ]]`,
				data:       responseTrue,
				log:        log,
			},
			want:          true,
			expectedError: "",
		},
		{
			name: "newEvaluateResponse_resultIsFalse",
			args: args{
				expression: `[[ eq .last_response.body.status "completed" ]]`,
				data:       responseFalse,
				log:        log,
			},
			want:          false,
			expectedError: "",
		},
		{
			name: "newEvaluateResponse_invalidExpressionError",
			args: args{
				expression: `eq .last_response.body.status "completed" ]]`,
				data:       responseFalse,
				log:        log,
			},
			want:          false,
			expectedError: "error while parsing boolean value of string: strconv.ParseBool: parsing \"eq .last_response.body.status \\\"completed\\\" ]]\": invalid syntax",
		},
		{
			name: "newEvaluateResponse_emptyExpressionError",
			args: args{
				expression: "",
				data:       responseFalse,
				log:        log,
			},
			want:          false,
			expectedError: "error while evaluating expression: the template result is empty",
		},
		{
			name: "newEvaluateResponse_incompleteExpressionError",
			args: args{
				expression: `[[.last_response.body.status]]`,
				data:       responseFalse,
				log:        log,
			},
			want:          false,
			expectedError: "error while parsing boolean value of string: strconv.ParseBool: parsing \"initiated\": invalid syntax",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expression := &valueTpl{}
			err := expression.Unpack(tt.args.expression)
			assert.NoError(t, err)

			got, err := evaluateResponse(expression, tt.args.data, tt.args.log)
			if err != nil {
				assert.EqualError(t, err, tt.expectedError)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestProcessExpression(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		// Cursor values.
		{in: ".first_response.foo", want: []string{"first_response", "foo"}},
		{in: ".first_response.", want: []string{"first_response", ""}},
		{in: ".last_response.foo", want: []string{"last_response", "foo"}},
		{in: ".last_response.", want: []string{"last_response", ""}},
		{in: ".parent_last_response.foo", want: []string{"parent_last_response", "foo"}},
		{in: ".parent_last_response.", want: []string{"parent_last_response", ""}},

		// Literal values.
		{in: ".literal_foo", want: []string{".literal_foo"}},
		{in: ".literal_foo.bar", want: []string{".literal_foo.bar"}},
		{in: "literal.foo.bar", want: []string{"literal.foo.bar"}},
		{in: "first_response.foo", want: []string{"first_response.foo"}},
		{in: ".first_response", want: []string{".first_response"}},
		{in: ".last_response", want: []string{".last_response"}},
		{in: ".parent_last_response", want: []string{".parent_last_response"}},
	}
	for _, test := range tests {
		got := processExpression(test.in)
		assert.Equal(t, test.want, got)
	}
}
