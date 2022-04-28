// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestNewSet(t *testing.T) {
	cases := []struct {
		name           string
		constructor    constructor
		config         map[string]interface{}
		expectedTarget targetInfo
		expectedErr    string
	}{
		{
			name:        "newSetResponse targets body",
			constructor: newSetResponse,
			config: map[string]interface{}{
				"target": "body.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "body"},
		},
		{
			name:        "newSetResponse targets something else",
			constructor: newSetResponse,
			config: map[string]interface{}{
				"target": "cursor.foo",
			},
			expectedErr: "invalid target: cursor.foo",
		},
		{
			name:        "newSetRequestPagination targets body",
			constructor: newSetRequestPagination,
			config: map[string]interface{}{
				"target": "body.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "body"},
		},
		{
			name:        "newSetRequestPagination targets header",
			constructor: newSetRequestPagination,
			config: map[string]interface{}{
				"target": "header.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "header"},
		},
		{
			name:        "newSetRequestPagination targets url param",
			constructor: newSetRequestPagination,
			config: map[string]interface{}{
				"target": "url.params.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "url.params"},
		},
		{
			name:        "newSetRequestPagination targets url value",
			constructor: newSetRequestPagination,
			config: map[string]interface{}{
				"target": "url.value",
			},
			expectedTarget: targetInfo{Type: "url.value"},
		},
		{
			name:        "newSetRequestPagination targets something else",
			constructor: newSetRequestPagination,
			config: map[string]interface{}{
				"target": "cursor.foo",
			},
			expectedErr: "invalid target: cursor.foo",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cfg := common.MustNewConfigFrom(tc.config)
			gotSet, gotErr := tc.constructor(cfg, nil)
			if tc.expectedErr == "" {
				assert.NoError(t, gotErr)
				assert.Equal(t, tc.expectedTarget, (gotSet.(*set)).targetInfo)
			} else {
				assert.EqualError(t, gotErr, tc.expectedErr)
			}
		})
	}
}

func TestSetFunctions(t *testing.T) {
	cases := []struct {
		name        string
		tfunc       func(ctx *transformContext, transformable transformable, key string, val interface{}) error
		paramCtx    *transformContext
		paramTr     transformable
		paramKey    string
		paramVal    string
		expectedTr  transformable
		expectedErr error
	}{
		{
			name:        "setBody",
			tfunc:       setBody,
			paramCtx:    &transformContext{},
			paramTr:     transformable{"body": mapstr.M{}},
			paramKey:    "a_key",
			paramVal:    "a_value",
			expectedTr:  transformable{"body": mapstr.M{"a_key": "a_value"}},
			expectedErr: nil,
		},
		{
			name:        "setHeader",
			tfunc:       setHeader,
			paramCtx:    &transformContext{},
			paramTr:     transformable{"header": http.Header{}},
			paramKey:    "a_key",
			paramVal:    "a_value",
			expectedTr:  transformable{"header": http.Header{"A_key": []string{"a_value"}}},
			expectedErr: nil,
		},
		{
			name:        "setURLParams",
			tfunc:       setURLParams,
			paramCtx:    &transformContext{},
			paramTr:     transformable{"url": newURL("http://foo.example.com")},
			paramKey:    "a_key",
			paramVal:    "a_value",
			expectedTr:  transformable{"url": newURL("http://foo.example.com?a_key=a_value")},
			expectedErr: nil,
		},
		{
			name:        "setURLValue",
			tfunc:       setURLValue,
			paramCtx:    &transformContext{},
			paramTr:     transformable{"url": newURL("http://foo.example.com")},
			paramVal:    "http://different.example.com",
			expectedTr:  transformable{"url": newURL("http://different.example.com")},
			expectedErr: nil,
		},
	}

	for _, tcase := range cases {
		tcase := tcase
		t.Run(tcase.name, func(t *testing.T) {
			gotErr := tcase.tfunc(tcase.paramCtx, tcase.paramTr, tcase.paramKey, tcase.paramVal)
			if tcase.expectedErr == nil {
				assert.NoError(t, gotErr)
			} else {
				assert.EqualError(t, gotErr, tcase.expectedErr.Error())
			}
			assert.EqualValues(t, tcase.expectedTr, tcase.paramTr)
		})
	}
}

func TestDifferentSetValueTypes(t *testing.T) {
	c1 := map[string]interface{}{
		"target":     "body.p1",
		"value":      `{"param":"value"}`,
		"value_type": "json",
	}

	cfg, err := common.NewConfigFrom(c1)
	require.NoError(t, err)

	transform, err := newSetResponse(cfg, logp.NewLogger("test"))
	require.NoError(t, err)

	testAppend := transform.(*set) //nolint:errcheck // Bad linter! Panic is a check.

	trCtx := emptyTransformContext()
	tr := transformable{}

	tr, err = testAppend.run(trCtx, tr)
	require.NoError(t, err)

	exp := mapstr.M{
		"p1": map[string]interface{}{
			"param": "value",
		},
	}

	assert.EqualValues(t, exp, tr.body())

	c2 := map[string]interface{}{
		"target":     "body.p1",
		"value":      "1",
		"value_type": "int",
	}

	cfg, err = common.NewConfigFrom(c2)
	require.NoError(t, err)

	transform, err = newSetResponse(cfg, logp.NewLogger("test"))
	require.NoError(t, err)

	testAppend = transform.(*set) //nolint:errcheck // Bad linter! Panic is a check.

	tr = transformable{}

	tr, err = testAppend.run(trCtx, tr)
	require.NoError(t, err)

	exp = mapstr.M{
		"p1": int64(1),
	}

	assert.EqualValues(t, exp, tr.body())

	c2["value_type"] = "string"

	cfg, err = common.NewConfigFrom(c2)
	require.NoError(t, err)

	transform, err = newSetResponse(cfg, logp.NewLogger("test"))
	require.NoError(t, err)

	testAppend = transform.(*set) //nolint:errcheck // Bad linter! Panic is a check.

	tr = transformable{}

	tr, err = testAppend.run(trCtx, tr)
	require.NoError(t, err)

	exp = mapstr.M{
		"p1": "1",
	}

	assert.EqualValues(t, exp, tr.body())
}

func newURL(u string) *url.URL {
	url, _ := url.Parse(u)
	return url
}
