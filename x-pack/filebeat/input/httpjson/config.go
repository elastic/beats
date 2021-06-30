// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
)

// config contains information about httpjson configuration
type config struct {
	OAuth2               *oauth2Config     `config:"oauth2"`
	APIKey               string            `config:"api_key"`
	AuthenticationScheme string            `config:"authentication_scheme"`
	HTTPClientTimeout    time.Duration     `config:"http_client_timeout"`
	HTTPHeaders          common.MapStr     `config:"http_headers"`
	HTTPMethod           string            `config:"http_method" validate:"required"`
	HTTPRequestBody      common.MapStr     `config:"http_request_body"`
	Interval             time.Duration     `config:"interval"`
	JSONObjects          string            `config:"json_objects_array"`
	SplitEventsBy        string            `config:"split_events_by"`
	NoHTTPBody           bool              `config:"no_http_body"`
	Pagination           *paginationConfig `config:"pagination"`
	RateLimit            *rateLimitConfig  `config:"rate_limit"`
	RetryMax             int               `config:"retry.max_attempts"`
	RetryWaitMin         time.Duration     `config:"retry.wait_min"`
	RetryWaitMax         time.Duration     `config:"retry.wait_max"`
	URL                  *urlConfig        `config:"url" validate:"required"`
	DateCursor           *dateCursorConfig `config:"date_cursor"`

	Transport httpcommon.HTTPTransportSettings `config:",inline"`
}

// Pagination contains information about httpjson pagination settings
type paginationConfig struct {
	Enabled          *bool         `config:"enabled"`
	ExtraBodyContent common.MapStr `config:"extra_body_content"`
	Header           *headerConfig `config:"header"`
	IDField          string        `config:"id_field"`
	RequestField     string        `config:"req_field"`
	URLField         string        `config:"url_field"`
	URL              string        `config:"url"`
}

// IsEnabled returns true if the `enable` field is set to true in the yaml.
func (p *paginationConfig) isEnabled() bool {
	return p != nil && (p.Enabled == nil || *p.Enabled)
}

// HTTP Header information for pagination
type headerConfig struct {
	FieldName    string         `config:"field_name" validate:"required"`
	RegexPattern *regexp.Regexp `config:"regex_pattern" validate:"required"`
}

// HTTP Header Rate Limit information
type rateLimitConfig struct {
	Limit     string `config:"limit"`
	Reset     string `config:"reset"`
	Remaining string `config:"remaining"`
}

type dateCursorConfig struct {
	Enabled         *bool           `config:"enabled"`
	Field           string          `config:"field"`
	URLField        string          `config:"url_field" validate:"required"`
	ValueTemplate   *templateConfig `config:"value_template"`
	DateFormat      string          `config:"date_format"`
	InitialInterval time.Duration   `config:"initial_interval"`
}

type templateConfig struct {
	*template.Template
}

func (t *templateConfig) Unpack(in string) error {
	tpl, err := template.New("tpl").Parse(in)
	if err != nil {
		return err
	}

	*t = templateConfig{Template: tpl}

	return nil
}

type urlConfig struct {
	*url.URL
}

func (u *urlConfig) Unpack(in string) error {
	parsed, err := url.Parse(in)
	if err != nil {
		return err
	}

	*u = urlConfig{URL: parsed}

	return nil
}

// IsEnabled returns true if the `enable` field is set to true in the yaml.
func (dc *dateCursorConfig) isEnabled() bool {
	return dc != nil && (dc.Enabled == nil || *dc.Enabled)
}

// IsEnabled returns true if the `enable` field is set to true in the yaml.
func (dc *dateCursorConfig) getDateFormat() string {
	if dc.DateFormat == "" {
		return time.RFC3339
	}
	return dc.DateFormat
}

func (dc *dateCursorConfig) Validate() error {
	if dc.DateFormat == "" {
		return nil
	}

	const knownTimestamp = 1602601228 // 2020-10-13T15:00:28+00:00 RFC3339
	knownDate := time.Unix(knownTimestamp, 0).UTC()

	dateStr := knownDate.Format(dc.DateFormat)
	if _, err := time.Parse(dc.DateFormat, dateStr); err != nil {
		return errors.New("invalid configuration: date_format is not a valid date layout")
	}

	return nil
}

func (c *config) Validate() error {
	switch strings.ToUpper(c.HTTPMethod) {
	case "GET", "POST":
		break
	default:
		return fmt.Errorf("httpjson input: Invalid http_method, %s", c.HTTPMethod)
	}
	if c.NoHTTPBody {
		if len(c.HTTPRequestBody) > 0 {
			return errors.New("invalid configuration: both no_http_body and http_request_body cannot be set simultaneously")
		}
		if c.Pagination != nil && (len(c.Pagination.ExtraBodyContent) > 0 || c.Pagination.RequestField != "") {
			return errors.New("invalid configuration: both no_http_body and pagination.extra_body_content or pagination.req_field cannot be set simultaneously")
		}
	}
	if c.Pagination != nil {
		if c.Pagination.Header != nil {
			if c.Pagination.RequestField != "" || c.Pagination.IDField != "" || len(c.Pagination.ExtraBodyContent) > 0 {
				return errors.New("invalid configuration: both pagination.header and pagination.req_field or pagination.id_field or pagination.extra_body_content cannot be set simultaneously")
			}
		}
	}
	if c.OAuth2.isEnabled() {
		if c.APIKey != "" || c.AuthenticationScheme != "" {
			return errors.New("invalid configuration: oauth2 and api_key or authentication_scheme cannot be set simultaneously")
		}
	}
	return nil
}

func newDefaultConfig() config {
	var c config
	c.HTTPMethod = "GET"
	c.HTTPClientTimeout = 60 * time.Second
	c.RetryWaitMin = 1 * time.Second
	c.RetryWaitMax = 60 * time.Second
	c.RetryMax = 5
	return c
}
