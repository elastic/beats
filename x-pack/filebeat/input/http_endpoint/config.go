// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"encoding/json"
	"errors"

	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
)

// Config contains information about httpjson configuration
type config struct {
	TLS           *tlscommon.ServerConfig `config:"ssl"`
	BasicAuth     bool                    `config:"basic_auth"`
	Username      string                  `config:"username"`
	Password      string                  `config:"password"`
	ResponseCode  int                     `config:"response_code" validate:"positive"`
	ResponseBody  string                  `config:"response_body"`
	ListenAddress string                  `config:"listen_address"`
	ListenPort    string                  `config:"listen_port"`
	URL           string                  `config:"url"`
	Prefix        string                  `config:"prefix"`
	ContentType   string                  `config:"content_type"`
	SecretHeader  string                  `config:"secret.header"`
	SecretValue   string                  `config:"secret.value"`
}

func defaultConfig() config {
	return config{
		BasicAuth:     false,
		Username:      "",
		Password:      "",
		ResponseCode:  200,
		ResponseBody:  `{"message": "success"}`,
		ListenAddress: "127.0.0.1",
		ListenPort:    "8000",
		URL:           "/",
		Prefix:        "json",
		ContentType:   "application/json",
		SecretHeader:  "",
		SecretValue:   "",
	}
}

func (c *config) Validate() error {
	if !json.Valid([]byte(c.ResponseBody)) {
		return errors.New("response_body must be valid JSON")
	}

	if c.BasicAuth {
		if c.Username == "" || c.Password == "" {
			return errors.New("Username and password required when basicauth is enabled")
		}
	}

	if (c.SecretHeader != "" && c.SecretValue == "") || (c.SecretHeader == "" && c.SecretValue != "") {
		return errors.New("Both secret.header and secret.value must be set")
	}

	return nil
}
