// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/management/status"
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
			gotAppend, gotErr := tc.constructor(cfg, noopReporter{}, nil)
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

var appendTemplateTests = []struct {
	name     string
	cfg      map[string]any
	ctx      *transformContext
	new      func(*conf.C, status.StatusReporter, *logp.Logger) (transform, error)
	in       map[string]any
	want     transformable
	wantErr  error
	wantStat []string
}{
	{
		name: "hello",
		new:  newAppendPagination,
		cfg: map[string]any{
			"target":     "body.dst",
			"value":      `[[printf "hello"]]`,
			"value_type": "string",
		},
		in: map[string]any{},
		want: transformable{
			"body": mapstr.M{"dst": []any{"hello"}},
		},
	},
	{
		name: "empty_no_default",
		new:  newAppendPagination,
		cfg: map[string]any{
			"target":             "body.dst",
			"value":              ``,
			"value_type":         "string",
			"do_not_log_failure": true,
		},
		in:   map[string]any{},
		want: transformable{},
	},
	{
		name: "empty_no_default_fail_on_error",
		new:  newAppendPagination,
		cfg: map[string]any{
			"target":                 "body.dst",
			"value":                  ``,
			"value_type":             "string",
			"fail_on_template_error": true,
			"do_not_log_failure":     true,
		},
		in:       map[string]any{},
		want:     transformable{},
		wantErr:  errEmptyTemplateResult,
		wantStat: nil,
	},
	{
		name: "empty_no_default_fail_on_error_empty_not_ok",
		new:  newAppendPagination,
		cfg: map[string]any{
			"target":                 "body.dst",
			"value":                  ``,
			"value_type":             "string",
			"fail_on_template_error": true,
			"do_not_log_failure":     false,
		},
		in:      map[string]any{},
		want:    transformable{},
		wantErr: errEmptyTemplateResult,
		wantStat: []string{
			"Degraded: failed to execute template dst: the template result is empty",
		},
	},
}

func TestAppendTemplate(t *testing.T) {
	for _, test := range appendTemplateTests {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := conf.NewConfigFrom(test.cfg)
			if err != nil {
				t.Fatalf("invalid template config: %v", err)
			}
			var stat testStatus
			tx, err := test.new(cfg, &stat, logp.NewLogger("test:"+test.name))
			if err != nil {
				t.Fatalf("failed to make append: %v", err)
			}
			btx, ok := tx.(basicTransform) // wat‽
			if !ok {
				t.Fatalf("transform is not a basicTransform: %T", tx)
			}
			ctx := test.ctx
			if ctx == nil {
				ctx = emptyTransformContext()
			}
			got, err := btx.run(ctx, test.in)
			if !sameError(err, test.wantErr) {
				t.Errorf("unexpected error: got=%q want=%q", err, test.wantErr)
			}
			if !cmp.Equal(stat.updates, test.wantStat) {
				t.Errorf("unexpected status updates: got=%q want=%q", stat.updates, test.wantStat)
			}
			if !cmp.Equal(got, test.want) {
				t.Errorf("unexpected status updates: got + want -\n%s", cmp.Diff(test.want, got))
			}
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

	transform, err := newAppendResponse(cfg, noopReporter{}, logp.NewLogger("test"))
	require.NoError(t, err)

	testAppend := transform.(*appendt)

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

	transform, err = newAppendResponse(cfg, noopReporter{}, logp.NewLogger("test"))
	require.NoError(t, err)

	testAppend = transform.(*appendt)

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

	transform, err = newAppendResponse(cfg, noopReporter{}, logp.NewLogger("test"))
	require.NoError(t, err)

	testAppend = transform.(*appendt)

	tr = transformable{}

	tr, err = testAppend.run(trCtx, tr)
	require.NoError(t, err)

	exp = mapstr.M{
		"p1": []interface{}{"1"},
	}

	assert.EqualValues(t, exp, tr.body())
}
