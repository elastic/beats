// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package v2

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
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
			name:        "newSetRequest targets body",
			constructor: newSetRequest,
			config: map[string]interface{}{
				"target": "body.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "body"},
		},
		{
			name:        "newSetRequest targets header",
			constructor: newSetRequest,
			config: map[string]interface{}{
				"target": "header.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "header"},
		},
		{
			name:        "newSetRequest targets url param",
			constructor: newSetRequest,
			config: map[string]interface{}{
				"target": "url.params.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "url.params"},
		},
		{
			name:        "newSetRequest targets something else",
			constructor: newSetRequest,
			config: map[string]interface{}{
				"target": "cursor.foo",
			},
			expectedErr: "invalid target: cursor.foo",
		},
		{
			name:        "newSetPagination targets body",
			constructor: newSetPagination,
			config: map[string]interface{}{
				"target": "body.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "body"},
		},
		{
			name:        "newSetPagination targets header",
			constructor: newSetPagination,
			config: map[string]interface{}{
				"target": "header.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "header"},
		},
		{
			name:        "newSetPagination targets url param",
			constructor: newSetPagination,
			config: map[string]interface{}{
				"target": "url.params.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "url.params"},
		},
		{
			name:        "newSetPagination targets url value",
			constructor: newSetPagination,
			config: map[string]interface{}{
				"target": "url.value",
			},
			expectedTarget: targetInfo{Type: "url.value"},
		},
		{
			name:        "newSetPagination targets something else",
			constructor: newSetPagination,
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
		tfunc       func(ctx *transformContext, transformable transformable, key, val string) error
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
			paramTr:     transformable{"body": common.MapStr{}},
			paramKey:    "a_key",
			paramVal:    "a_value",
			expectedTr:  transformable{"body": common.MapStr{"a_key": "a_value"}},
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

func newURL(u string) url.URL {
	url, _ := url.Parse(u)
	return *url
}
