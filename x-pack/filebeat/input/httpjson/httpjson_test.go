// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"net/http"
	"regexp"
	"strconv"
	"testing"
	"time"

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
	config := &Pagination{
		IDField:          "id",
		RequestField:     "pagination_id",
		ExtraBodyContent: extraBodyContent,
		URL:              "https://test-123",
	}
	ri, err := createRequestInfoFromBody(
		config,
		common.MapStr(m),
		common.MapStr(m),
		&requestInfo{
			url:        "",
			contentMap: common.MapStr{},
			headers:    common.MapStr{},
		},
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

// Test getRateLimit function with a remaining quota, expect to receive 0, nil.
func TestGetRateLimitCase1(t *testing.T) {
	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "120")
	header.Add("X-Rate-Limit-Remaining", "118")
	header.Add("X-Rate-Limit-Reset", "1581658643")
	rateLimit := &RateLimit{
		Limit:     "X-Rate-Limit-Limit",
		Reset:     "X-Rate-Limit-Reset",
		Remaining: "X-Rate-Limit-Remaining",
	}
	epoch, err := getRateLimit(header, rateLimit)
	if err != nil || epoch != 0 {
		t.Fatal("Failed to test getRateLimit.")
	}
}

// Test getRateLimit function with a past time, expect to receive 0, nil.
func TestGetRateLimitCase2(t *testing.T) {
	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "10")
	header.Add("X-Rate-Limit-Remaining", "0")
	header.Add("X-Rate-Limit-Reset", "1581658643")
	rateLimit := &RateLimit{
		Limit:     "X-Rate-Limit-Limit",
		Reset:     "X-Rate-Limit-Reset",
		Remaining: "X-Rate-Limit-Remaining",
	}
	epoch, err := getRateLimit(header, rateLimit)
	if err != nil || epoch != 0 {
		t.Fatal("Failed to test getRateLimit.")
	}
}

// Test getRateLimit function with a time yet to come, expect to receive <reset-value>, nil.
func TestGetRateLimitCase3(t *testing.T) {
	epoch := time.Now().Unix() + 100
	header := make(http.Header)
	header.Add("X-Rate-Limit-Limit", "10")
	header.Add("X-Rate-Limit-Remaining", "0")
	header.Add("X-Rate-Limit-Reset", strconv.FormatInt(epoch, 10))
	rateLimit := &RateLimit{
		Limit:     "X-Rate-Limit-Limit",
		Reset:     "X-Rate-Limit-Reset",
		Remaining: "X-Rate-Limit-Remaining",
	}
	epoch2, err := getRateLimit(header, rateLimit)
	if err != nil || epoch2 != epoch {
		t.Fatal("Failed to test getRateLimit.")
	}
}
