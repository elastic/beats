// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
)

// Config contains information about httpjson configuration
type config struct {
	APIKey               string            `config:"api_key"`
	AuthenticationScheme string            `config:"authentication_scheme"`
	HTTPClientTimeout    time.Duration     `config:"http_client_timeout"`
	HTTPHeaders          common.MapStr     `config:"http_headers"`
	HTTPMethod           string            `config:"http_method" validate:"required"`
	HTTPRequestBody      common.MapStr     `config:"http_request_body"`
	Interval             time.Duration     `config:"interval"`
	JSONObjects          string            `config:"json_objects_array"`
	NoHTTPBody           bool              `config:"no_http_body"`
	Pagination           *Pagination       `config:"pagination"`
	RateLimit            *RateLimit        `config:"rate_limit"`
	TLS                  *tlscommon.Config `config:"ssl"`
	URL                  string            `config:"url" validate:"required"`
}

// Pagination contains information about httpjson pagination settings
type Pagination struct {
	Enabled          *bool         `config:"enabled"`
	ExtraBodyContent common.MapStr `config:"extra_body_content"`
	Header           *Header       `config:"header"`
	IDField          string        `config:"id_field"`
	RequestField     string        `config:"req_field"`
	URL              string        `config:"url"`
}

// IsEnabled returns true if the `enable` field is set to true in the yaml.
func (p *Pagination) IsEnabled() bool {
	return p != nil && (p.Enabled == nil || *p.Enabled)
}

// HTTP Header information for pagination
type Header struct {
	FieldName    string         `config:"field_name" validate:"required"`
	RegexPattern *regexp.Regexp `config:"regex_pattern" validate:"required"`
}

// HTTP Header Rate Limit information
type RateLimit struct {
	Limit     string `config:"limit"`
	Reset     string `config:"reset"`
	Remaining string `config:"remaining"`
}

func (c *config) Validate() error {
	switch strings.ToUpper(c.HTTPMethod) {
	case "GET":
		break
	case "POST":
		break
	default:
		return errors.Errorf("httpjson input: Invalid http_method, %s", c.HTTPMethod)
	}
	if c.NoHTTPBody {
		if len(c.HTTPRequestBody) > 0 {
			return errors.Errorf("invalid configuration: both no_http_body and http_request_body cannot be set simultaneously")
		}
		if c.Pagination != nil && (len(c.Pagination.ExtraBodyContent) > 0 || c.Pagination.RequestField != "") {
			return errors.Errorf("invalid configuration: both no_http_body and pagination.extra_body_content or pagination.req_field cannot be set simultaneously")
		}
	}
	if c.Pagination != nil {
		if c.Pagination.Header != nil {
			if c.Pagination.RequestField != "" || c.Pagination.IDField != "" || len(c.Pagination.ExtraBodyContent) > 0 {
				return errors.Errorf("invalid configuration: both pagination.header and pagination.req_field or pagination.id_field or pagination.extra_body_content cannot be set simultaneously")
			}
		}
	}
	return nil
}

func defaultConfig() config {
	var c config
	c.HTTPMethod = "GET"
	c.HTTPClientTimeout = 60 * time.Second
	return c
}
