// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"context"
	"fmt"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	beattest "github.com/elastic/beats/v7/libbeat/publisher/testing"
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

	cfg := common.MustNewConfigFrom(map[string]interface{}{
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

	requestFactory := newRequestFactory(config.Request, config.Chain, nil, log)
	pagination := newPagination(config, client, log)
	responseProcessor := newResponseProcessor(config.Response, pagination, log)

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

func TestJsonNorm(t *testing.T) {
	type args struct {
		sr   string
		bNew interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "will return string value from interface",
			args: args{
				sr: "a",
				bNew: map[string]interface{}{
					"a": "a_value",
				},
			},
			want:    "a_value",
			wantErr: false,
		},
		{
			name: "will return interface value from interface",
			args: args{
				sr: "a",
				bNew: map[string]interface{}{
					"a": map[string]interface{}{
						"b": "b_value",
					},
				},
			},
			want: map[string]interface{}{
				"b": "b_value",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jsonNorm(tt.args.sr, tt.args.bNew)
			if (err != nil) != tt.wantErr {
				t.Errorf("jsonNorm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("jsonNorm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJsonArr(t *testing.T) {
	type args struct {
		sr   string
		bNew interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    []interface{}
		wantErr bool
	}{
		{
			name: "will return array of interface values from interface",
			args: args{
				sr: "a",
				bNew: []interface{}{
					map[string]interface{}{
						"a": "a_value_1",
					},
					map[string]interface{}{
						"a": "a_value_2",
					},
				},
			},
			want: []interface{}{
				"a_value_1",
				"a_value_2",
			},
			wantErr: false,
		},
		{
			name: "will return array of embedded interface values from interface",
			args: args{
				sr: "a",
				bNew: []interface{}{
					map[string]interface{}{
						"a": map[string]interface{}{
							"b": "b_value",
						},
					},
					map[string]interface{}{
						"a": map[string]interface{}{
							"b": "b_value",
						},
					},
				},
			},
			want: []interface{}{
				map[string]interface{}{
					"b": "b_value",
				},
				map[string]interface{}{
					"b": "b_value",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jsonArr(tt.args.sr, tt.args.bNew)
			if (err != nil) != tt.wantErr {
				t.Errorf("jsonArr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("jsonArr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetJSON(t *testing.T) {
	type args struct {
		b   string
		str string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "will return string value from string json",
			args: args{
				b:   `{"a": "a_value"}`,
				str: "a",
			},
			want:    []string{"a_value"},
			wantErr: false,
		},
		{
			name: "will return string value from string embedded json",
			args: args{
				b:   `{"a": {"b": "b_value"}}`,
				str: "a.b",
			},
			want:    []string{"b_value"},
			wantErr: false,
		},
		{
			name: "will return string value from string embedded Array json",
			args: args{
				b:   `{"a": [{"b": "b_value_1"},{"b": "b_value_2"},{"b": "b_value_3"}]}`,
				str: "a.#.b",
			},
			want:    []string{"b_value_1", "b_value_2", "b_value_3"},
			wantErr: false,
		},
		{
			name: "will return string value from string Array json",
			args: args{
				b:   `[{"b": "b_value_1"},{"b": "b_value_2"},{"b": "b_value_3"}]`,
				str: "#.b",
			},
			want:    []string{"b_value_1", "b_value_2", "b_value_3"},
			wantErr: false,
		},
		{
			name: "will return string value from string embedded Array json",
			args: args{
				b:   `{"a":[{"b":{"c":"c_value_1"}},{"b":{"c":"c_value_2"}},{"b":{"c":"c_value_3"}}]}`,
				str: "a.#.b.c",
			},
			want:    []string{"c_value_1", "c_value_2", "c_value_3"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getJSON(tt.args.b, tt.args.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("getJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJsonInterface(t *testing.T) {
	type args struct {
		str    string
		comStr string
		bNew   []byte
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "will return interface value from bytes",
			args: args{
				str:    "{",
				comStr: "a",
				bNew:   []byte(`{"a":"a_value"}`),
			},
			want: map[string]interface{}{
				"a": "a_value",
			},
			wantErr: false,
		},
		{
			name: "will return embedded interface value from bytes",
			args: args{
				str:    "{",
				comStr: "a",
				bNew:   []byte(`{"a": {"b": "b_value"}}`),
			},
			want: map[string]interface{}{
				"a": map[string]interface{}{
					"b": "b_value",
				},
			},
			wantErr: false,
		},
		{
			name: "can not use # if json value is normal json",
			args: args{
				str:    "{",
				comStr: "#",
				bNew:   []byte(`{"a": {"b": "b_value"}}`),
			},
			wantErr: true,
		},
		{
			name: "will return []interface{} if value is array of json",
			args: args{
				str:    "[",
				comStr: "#",
				bNew:   []byte(`[{"b": "b_value_1"},{"b": "b_value_2"},{"b": "b_value_3"}]`),
			},
			want: []interface{}{
				map[string]interface{}{
					"b": "b_value_1",
				},
				map[string]interface{}{
					"b": "b_value_2",
				},
				map[string]interface{}{
					"b": "b_value_3",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jsonInterface(tt.args.str, tt.args.comStr, tt.args.bNew)
			if (err != nil) != tt.wantErr {
				t.Errorf("jsonInterface() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("jsonInterface() = %v, want %v", got, tt.want)
			}
		})
	}
}
