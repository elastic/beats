// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
)

// Config contains information about httpjson configuration
type config struct {
	APIKey            string            `config:"api_key"`
	HTTPClientTimeout time.Duration     `config:"http_client_timeout"`
	HTTPMethod        string            `config:"http_method" validate:"required"`
	HTTPRequestBody   common.MapStr     `config:"http_request_body"`
	Interval          time.Duration     `config:"interval" validate:"required"`
	JSONObjects       string            `config:"json_objects_array"`
	Pagination        *Pagination       `config:"pagination"`
	TLS               *tlscommon.Config `config:"ssl"`
	URL               string            `config:"url" validate:"required"`
}

// Pagination contains information about httpjson pagination settings
type Pagination struct {
	IsEnabled        bool          `config:"enabled"`
	ExtraBodyContent common.MapStr `config:"extra_body_content"`
	IDField          string        `config:"id_field"`
	RequestField     string        `config:"req_field"`
	URL              string        `config:"url"`
}

func (c *config) Validate() error {
	if c.Interval < 3600*time.Second && c.Interval != 0 {
		return errors.New("httpjson input: interval must not be less than 3600 seconds - ")
	}
	switch strings.ToUpper(c.HTTPMethod) {
	case "GET":
		break
	case "POST":
		break
	default:
		return errors.New(fmt.Sprintf("httpjson input: Invalid http_method, %s - ", c.HTTPMethod))
	}
	return nil
}

func defaultConfig() config {
	var c config
	c.HTTPMethod = "GET"
	c.HTTPClientTimeout = 60 * time.Second
	return c
}
