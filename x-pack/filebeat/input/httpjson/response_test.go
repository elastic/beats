// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common"
)

func TestTemplateValues(t *testing.T) {
	resp := &response{
		page: 1,
		url:  *(newURL("http://test?p1=v1")),
		header: http.Header{
			"Authorization": []string{"Bearer token"},
		},
		body: common.MapStr{
			"param": "value",
		},
	}

	vals := resp.templateValues()

	assert.Equal(t, resp.page, vals["page"])
	v, _ := vals.GetValue("url.value")
	assert.Equal(t, resp.url.String(), v)
	v, _ = vals.GetValue("url.params")
	assert.EqualValues(t, resp.url.Query(), v)
	assert.EqualValues(t, resp.header, vals["header"])
	assert.EqualValues(t, resp.body, vals["body"])

	resp = nil

	vals = resp.templateValues()

	assert.NotNil(t, vals)
	assert.Equal(t, 0, len(vals))
}
