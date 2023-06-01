// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"bytes"
	"context"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/logp"
)

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
			expectedError: "error while parsing boolean value of string : strconv.ParseBool: parsing \"eq .last_response.body.status \\\"completed\\\" ]]\": invalid syntax",
		},
		{
			name: "newEvaluateResponse_emptyExpressionError",
			args: args{
				expression: "",
				data:       responseFalse,
				log:        log,
			},
			want:          false,
			expectedError: "error while evaluating expression : the template result is empty",
		},
		{
			name: "newEvaluateResponse_incompleteExpressionError",
			args: args{
				expression: `[[.last_response.body.status]]`,
				data:       responseFalse,
				log:        log,
			},
			want:          false,
			expectedError: "error while parsing boolean value of string : strconv.ParseBool: parsing \"initiated\": invalid syntax",
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
