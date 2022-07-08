// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"bytes"
	"context"
	"crypto/x509"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/stretchr/testify/assert"
)

func TestPolicy_CustomRetryPolicy(t *testing.T) {
	statusCompleted := `{"status":"completed"}`
	statusInitiated := `{"status":"cmnsmc"}`

	exp := &valueTpl{}
	err := exp.Unpack(`[[ eq .last_response.body.status "completed" ]]`)
	assert.NoError(t, err)

	expErr := &valueTpl{}
	err = exp.Unpack("")
	assert.NoError(t, err)
	type fields struct {
		fn         Evaluate
		expression *valueTpl
		log        *logp.Logger
	}
	type args struct {
		ctx  context.Context
		resp *http.Response
		err  error
	}
	tests := []struct {
		name          string
		fields        fields
		args          args
		want          bool
		expectedError string
	}{
		{
			name: "customRetryPolicy_doNotRetryFurther",
			fields: fields{
				fn:         evaluateResponse,
				expression: exp,
				log:        logp.NewLogger(""),
			},
			args: args{
				ctx:  context.Background(),
				resp: getTestResponse(statusCompleted, 200),
				err:  nil,
			},
			want:          false,
			expectedError: "",
		},
		{
			name: "customRetryPolicy_keepRetrying",
			fields: fields{
				fn:         evaluateResponse,
				expression: exp,
				log:        logp.NewLogger(""),
			},
			args: args{
				ctx:  context.Background(),
				resp: getTestResponse(statusInitiated, 200),
				err:  nil,
			},
			want:          true,
			expectedError: "",
		},
		{
			name: "customRetryPolicy_emptyTemplateError",
			fields: fields{
				fn:         evaluateResponse,
				expression: expErr,
				log:        logp.NewLogger(""),
			},
			args: args{
				ctx:  context.Background(),
				resp: getTestResponse(statusCompleted, 200),
				err:  nil,
			},
			want:          false,
			expectedError: "error while evaluating expression : the template result is empty",
		},
		{
			name: "customRetryPolicy_internalServerError",
			fields: fields{
				fn:         evaluateResponse,
				expression: exp,
				log:        logp.NewLogger(""),
			},
			args: args{
				ctx:  context.Background(),
				resp: getTestResponse(statusCompleted, 500),
				err:  nil,
			},
			want:          true,
			expectedError: "",
		},
		{
			name: "customRetryPolicy_unknownCertError",
			fields: fields{
				fn:         evaluateResponse,
				expression: exp,
				log:        logp.NewLogger(""),
			},
			args: args{
				ctx:  context.Background(),
				resp: getTestResponse(statusCompleted, 200),
				err:  &url.Error{Err: x509.UnknownAuthorityError{}},
			},
			want:          false,
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Policy{
				fn:         tt.fields.fn,
				expression: tt.fields.expression,
				log:        tt.fields.log,
			}
			got, err := p.CustomRetryPolicy(tt.args.ctx, tt.args.resp, tt.args.err)
			if err != nil {
				assert.Error(t, err, tt.expectedError)
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func getTestResponse(exprStr string, statusCode int) *http.Response {
	return &http.Response{
		StatusCode:    statusCode,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          io.NopCloser(bytes.NewBufferString(exprStr)),
		ContentLength: int64(len(string(exprStr))),
		Request:       nil,
		Header:        make(http.Header, 0),
	}
}
