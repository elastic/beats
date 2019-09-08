// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type config struct {
	APIKey                     string      `config:"api_key"`
	HTTPClientTimeout          int64       `config:"http_client_timeout"`
	HTTPMethod                 string      `config:"http_method" validate:"required"`
	HTTPRequestBody            interface{} `config:"http_request_body"`
	Interval                   int64       `config:"interval_in_seconds" validate:"required"`
	JSONObjects                string      `config:"json_objects_array"`
	PaginationEnable           bool        `config:"pagination_enable"`
	PaginationExtraBodyContent interface{} `config:"pagination_extra_body_content"`
	PaginationIdField          string      `config:"pagination_id_field"`
	PaginationRequestField     string      `config:"pagination_req_field"`
	PaginationURL              string      `config:"pagination_url"`
	ServerName                 string      `config:"server_name"`
	URL                        string      `config:"url" validate:"required"`
}

func (c *config) Validate() error {
	if c.Interval < 3600 && c.Interval != 0 {
		return errors.New("httpjson input: interval_in_seconds must not be less than 3600 - ")
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
	c.HTTPClientTimeout = 30
	return c
}
