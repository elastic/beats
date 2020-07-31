// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/elastic/beats/v7/libbeat/common"
)

func TestGetNextLinkFromHeader(t *testing.T) {
	header := make(http.Header)
	header.Add("Link", "<https://dev-168980.okta.com/api/v1/logs>; rel=\"self\"")
	header.Add("Link", "<https://dev-168980.okta.com/api/v1/logs?after=1581658181086_1>; rel=\"next\"")
	re, _ := regexp.Compile("<([^>]+)>; *rel=\"next\"(?:,|$)")
	url, err := getNextLinkFromHeader(header, "Link", re)
	if url != "https://dev-168980.okta.com/api/v1/logs?after=1581658181086_1" {
		t.Fatal("Failed to test getNextLinkFromHeader. URL " + url + " is not expected")
	}
	if err != nil {
		t.Fatal("Failed to test getNextLinkFromHeader with error:", err)
	}
}

func TestCreateRequestInfoFromBody(t *testing.T) {
	m := map[string]interface{}{
		"id": 100,
	}
	extraBodyContent := common.MapStr{"extra_body": "abc"}
	pagination := &pagination{
		idField:          "id",
		requestField:     "pagination_id",
		extraBodyContent: extraBodyContent,
		url:              "https://test-123",
	}
	ri := &requestInfo{
		url:        "",
		contentMap: common.MapStr{},
		headers:    common.MapStr{},
	}
	err := pagination.setRequestInfoFromBody(
		common.MapStr(m),
		common.MapStr(m),
		ri,
	)
	if ri.url != "https://test-123" {
		t.Fatal("Failed to test createRequestInfoFromBody. URL should be https://test-123.")
	}
	p, err := ri.contentMap.GetValue("pagination_id")
	if err != nil {
		t.Fatal("Failed to test createRequestInfoFromBody with error", err)
	}
	switch pt := p.(type) {
	case int:
		if pt != 100 {
			t.Fatalf("Failed to test createRequestInfoFromBody. pagination_id value %d should be 100.", pt)
		}
	default:
		t.Fatalf("Failed to test createRequestInfoFromBody. pagination_id value %T should be int.", pt)
	}
	b, err := ri.contentMap.GetValue("extra_body")
	if err != nil {
		t.Fatal("Failed to test createRequestInfoFromBody with error", err)
	}
	switch bt := b.(type) {
	case string:
		if bt != "abc" {
			t.Fatalf("Failed to test createRequestInfoFromBody. extra_body value %s does not match \"abc\".", bt)
		}
	default:
		t.Fatalf("Failed to test createRequestInfoFromBody. extra_body type %T should be string.", bt)
	}
}
