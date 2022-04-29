// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//nolint:dupl,goconst // Bad linter! Tests should be explicit and local.
package httpjson

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestNewAppend(t *testing.T) {
	cases := []struct {
		name           string
		constructor    constructor
		config         map[string]interface{}
		expectedTarget targetInfo
		expectedErr    string
	}{
		{
			name:        "newAppendResponse targets body",
			constructor: newAppendResponse,
			config: map[string]interface{}{
				"target": "body.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "body"},
		},
		{
			name:        "newAppendResponse targets something else",
			constructor: newAppendResponse,
			config: map[string]interface{}{
				"target": "cursor.foo",
			},
			expectedErr: "invalid target: cursor.foo",
		},
		{
			name:        "newAppendRequest targets body",
			constructor: newAppendRequest,
			config: map[string]interface{}{
				"target": "body.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "body"},
		},
		{
			name:        "newAppendRequest targets header",
			constructor: newAppendRequest,
			config: map[string]interface{}{
				"target": "header.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "header"},
		},
		{
			name:        "newAppendRequest targets url param",
			constructor: newAppendRequest,
			config: map[string]interface{}{
				"target": "url.params.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "url.params"},
		},
		{
			name:        "newAppendRequest targets something else",
			constructor: newAppendRequest,
			config: map[string]interface{}{
				"target": "cursor.foo",
			},
			expectedErr: "invalid target: cursor.foo",
		},
		{
			name:        "newAppendPagination targets body",
			constructor: newAppendPagination,
			config: map[string]interface{}{
				"target": "body.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "body"},
		},
		{
			name:        "newAppendPagination targets header",
			constructor: newAppendPagination,
			config: map[string]interface{}{
				"target": "header.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "header"},
		},
		{
			name:        "newAppendPagination targets url param",
			constructor: newAppendPagination,
			config: map[string]interface{}{
				"target": "url.params.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "url.params"},
		},
		{
			name:        "newAppendPagination targets url value",
			constructor: newAppendPagination,
			config: map[string]interface{}{
				"target": "url.value",
			},
			expectedErr: "invalid target type: url.value",
		},
		{
			name:        "newAppendPagination targets something else",
			constructor: newAppendPagination,
			config: map[string]interface{}{
				"target": "cursor.foo",
			},
			expectedErr: "invalid target: cursor.foo",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cfg := conf.MustNewConfigFrom(tc.config)
			gotAppend, gotErr := tc.constructor(cfg, nil)
			if tc.expectedErr == "" {
				assert.NoError(t, gotErr)
				assert.Equal(t, tc.expectedTarget, (gotAppend.(*appendt)).targetInfo)
			} else {
				assert.EqualError(t, gotErr, tc.expectedErr)
			}
		})
	}
}

func TestAppendFunctions(t *testing.T) {
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
			name:        "appendBody",
			tfunc:       appendBody,
			paramCtx:    &transformContext{},
			paramTr:     transformable{"body": mapstr.M{"a_key": "a_value"}},
			paramKey:    "a_key",
			paramVal:    "another_value",
			expectedTr:  transformable{"body": mapstr.M{"a_key": []interface{}{"a_value", "another_value"}}},
			expectedErr: nil,
		},
		{
			name:        "appendBodyWithSingleValue",
			tfunc:       appendBody,
			paramCtx:    &transformContext{},
			paramTr:     transformable{"body": mapstr.M{}},
			paramKey:    "a_key",
			paramVal:    "a_value",
			expectedTr:  transformable{"body": mapstr.M{"a_key": []interface{}{"a_value"}}},
			expectedErr: nil,
		},
		{
			name:     "appendHeader",
			tfunc:    appendHeader,
			paramCtx: &transformContext{},
			paramTr: transformable{"header": http.Header{
				"A_key": []string{"a_value"},
			}},
			paramKey:    "a_key",
			paramVal:    "another_value",
			expectedTr:  transformable{"header": http.Header{"A_key": []string{"a_value", "another_value"}}},
			expectedErr: nil,
		},
		{
			name:        "appendURLParams",
			tfunc:       appendURLParams,
			paramCtx:    &transformContext{},
			paramTr:     transformable{"url": newURL("http://foo.example.com?a_key=a_value")},
			paramKey:    "a_key",
			paramVal:    "another_value",
			expectedTr:  transformable{"url": newURL("http://foo.example.com?a_key=a_value&a_key=another_value")},
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

func TestDifferentAppendValueTypes(t *testing.T) {
	c1 := map[string]interface{}{
		"target":     "body.p1",
		"value":      `{"param":"value"}`,
		"value_type": "json",
	}

	cfg, err := conf.NewConfigFrom(c1)
	require.NoError(t, err)

	transform, err := newAppendResponse(cfg, logp.NewLogger("test"))
	require.NoError(t, err)

	testAppend := transform.(*appendt) //nolint:errcheck // Bad linter! Panic is a check.

	trCtx := emptyTransformContext()
	tr := transformable{}

	tr, err = testAppend.run(trCtx, tr)
	require.NoError(t, err)

	exp := mapstr.M{
		"p1": []interface{}{
			map[string]interface{}{
				"param": "value",
			},
		},
	}

	assert.EqualValues(t, exp, tr.body())

	c2 := map[string]interface{}{
		"target":     "body.p1",
		"value":      "1",
		"value_type": "int",
	}

	cfg, err = conf.NewConfigFrom(c2)
	require.NoError(t, err)

	transform, err = newAppendResponse(cfg, logp.NewLogger("test"))
	require.NoError(t, err)

	testAppend = transform.(*appendt) //nolint:errcheck // Bad linter! Panic is a check.

	tr = transformable{}

	tr, err = testAppend.run(trCtx, tr)
	require.NoError(t, err)

	exp = mapstr.M{
		"p1": []interface{}{int64(1)},
	}

	assert.EqualValues(t, exp, tr.body())

	c2["value_type"] = "string"

	cfg, err = conf.NewConfigFrom(c2)
	require.NoError(t, err)

	transform, err = newAppendResponse(cfg, logp.NewLogger("test"))
	require.NoError(t, err)

	testAppend = transform.(*appendt) //nolint:errcheck // Bad linter! Panic is a check.

	tr = transformable{}

	tr, err = testAppend.run(trCtx, tr)
	require.NoError(t, err)

	exp = mapstr.M{
		"p1": []interface{}{"1"},
	}

	assert.EqualValues(t, exp, tr.body())
}
