// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//nolint:dupl // Bad linter!
package httpjson

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestNewDelete(t *testing.T) {
	cases := []struct {
		name           string
		constructor    constructor
		config         map[string]interface{}
		expectedTarget targetInfo
		expectedErr    string
	}{
		{
			name:        "newDeleteResponse targets body",
			constructor: newDeleteResponse,
			config: map[string]interface{}{
				"target": "body.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "body"},
		},
		{
			name:        "newDeleteResponse targets something else",
			constructor: newDeleteResponse,
			config: map[string]interface{}{
				"target": "cursor.foo",
			},
			expectedErr: "invalid target: cursor.foo",
		},
		{
			name:        "newDeleteRequest targets body",
			constructor: newDeleteRequest,
			config: map[string]interface{}{
				"target": "body.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "body"},
		},
		{
			name:        "newDeleteRequest targets header",
			constructor: newDeleteRequest,
			config: map[string]interface{}{
				"target": "header.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "header"},
		},
		{
			name:        "newDeleteRequest targets url param",
			constructor: newDeleteRequest,
			config: map[string]interface{}{
				"target": "url.params.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "url.params"},
		},
		{
			name:        "newDeleteRequest targets something else",
			constructor: newDeleteRequest,
			config: map[string]interface{}{
				"target": "cursor.foo",
			},
			expectedErr: "invalid target: cursor.foo",
		},
		{
			name:        "newDeletePagination targets body",
			constructor: newDeletePagination,
			config: map[string]interface{}{
				"target": "body.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "body"},
		},
		{
			name:        "newDeletePagination targets header",
			constructor: newDeletePagination,
			config: map[string]interface{}{
				"target": "header.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "header"},
		},
		{
			name:        "newDeletePagination targets url param",
			constructor: newDeletePagination,
			config: map[string]interface{}{
				"target": "url.params.foo",
			},
			expectedTarget: targetInfo{Name: "foo", Type: "url.params"},
		},
		{
			name:        "newDeletePagination targets url value",
			constructor: newDeletePagination,
			config: map[string]interface{}{
				"target": "url.value",
			},
			expectedErr: "invalid target type: url.value",
		},
		{
			name:        "newDeletePagination targets something else",
			constructor: newDeletePagination,
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
			gotDelete, gotErr := tc.constructor(cfg, nil)
			if tc.expectedErr == "" {
				assert.NoError(t, gotErr)
				assert.Equal(t, tc.expectedTarget, (gotDelete.(*delete)).targetInfo)
			} else {
				assert.EqualError(t, gotErr, tc.expectedErr)
			}
		})
	}
}

func TestDeleteFunctions(t *testing.T) {
	cases := []struct {
		name        string
		tfunc       func(ctx *transformContext, transformable transformable, key string) error
		paramCtx    *transformContext
		paramTr     transformable
		paramKey    string
		expectedTr  transformable
		expectedErr error
	}{
		{
			name:        "deleteBody",
			tfunc:       deleteBody,
			paramCtx:    &transformContext{},
			paramTr:     transformable{"body": mapstr.M{"a_key": "a_value"}},
			paramKey:    "a_key",
			expectedTr:  transformable{"body": mapstr.M{}},
			expectedErr: nil,
		},
		{
			name:     "deleteHeader",
			tfunc:    deleteHeader,
			paramCtx: &transformContext{},
			paramTr: transformable{"header": http.Header{
				"A_key": []string{"a_value"},
			}},
			paramKey:    "a_key",
			expectedTr:  transformable{"header": http.Header{}},
			expectedErr: nil,
		},
		{
			name:        "deleteURLParams",
			tfunc:       deleteURLParams,
			paramCtx:    &transformContext{},
			paramTr:     transformable{"url": newURL("http://foo.example.com?a_key=a_value")},
			paramKey:    "a_key",
			expectedTr:  transformable{"url": newURL("http://foo.example.com")},
			expectedErr: nil,
		},
	}

	for _, tcase := range cases {
		tcase := tcase
		t.Run(tcase.name, func(t *testing.T) {
			gotErr := tcase.tfunc(tcase.paramCtx, tcase.paramTr, tcase.paramKey)
			if tcase.expectedErr == nil {
				assert.NoError(t, gotErr)
			} else {
				assert.EqualError(t, gotErr, tcase.expectedErr.Error())
			}
			assert.EqualValues(t, tcase.expectedTr, tcase.paramTr)
		})
	}
}
