// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"encoding/json"
	"errors"
	"net/textproto"

	"github.com/elastic/beats/v8/libbeat/common/transport/tlscommon"
)

// Config contains information about httpjson configuration
type config struct {
	TLS                   *tlscommon.ServerConfig `config:"ssl"`
	BasicAuth             bool                    `config:"basic_auth"`
	Username              string                  `config:"username"`
	Password              string                  `config:"password"`
	ResponseCode          int                     `config:"response_code" validate:"positive"`
	ResponseBody          string                  `config:"response_body"`
	ListenAddress         string                  `config:"listen_address"`
	ListenPort            string                  `config:"listen_port"`
	URL                   string                  `config:"url"`
	Prefix                string                  `config:"prefix"`
	ContentType           string                  `config:"content_type"`
	SecretHeader          string                  `config:"secret.header"`
	SecretValue           string                  `config:"secret.value"`
	HMACHeader            string                  `config:"hmac.header"`
	HMACKey               string                  `config:"hmac.key"`
	HMACType              string                  `config:"hmac.type"`
	HMACPrefix            string                  `config:"hmac.prefix"`
	IncludeHeaders        []string                `config:"include_headers"`
	PreserveOriginalEvent bool                    `config:"preserve_original_event"`
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
		HMACHeader:    "",
		HMACKey:       "",
		HMACType:      "",
		HMACPrefix:    "",
	}
}

func (c *config) Validate() error {
	if !json.Valid([]byte(c.ResponseBody)) {
		return errors.New("response_body must be valid JSON")
	}

	if c.BasicAuth {
		if c.Username == "" || c.Password == "" {
			return errors.New("username and password required when basicauth is enabled")
		}
	}

	if (c.SecretHeader != "" && c.SecretValue == "") || (c.SecretHeader == "" && c.SecretValue != "") {
		return errors.New("both secret.header and secret.value must be set")
	}

	if (c.HMACHeader != "" && c.HMACKey == "") || (c.HMACHeader == "" && c.HMACKey != "") {
		return errors.New("both hmac.header and hmac.key must be set")
	}

	if c.HMACType != "" && !(c.HMACType == "sha1" || c.HMACType == "sha256") {
		return errors.New("hmac.type must be sha1 or sha256")
	}

	return nil
}

func canonicalizeHeaders(headerConf []string) (includeHeaders []string) {
	for i := range headerConf {
		headerConf[i] = textproto.CanonicalMIMEHeaderKey(headerConf[i])
	}
	return headerConf
}
