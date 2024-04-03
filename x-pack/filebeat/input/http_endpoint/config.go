// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package http_endpoint

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/textproto"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

// Available providers for CRC validation (use lowercase)
// Constructor function as a value for each provider
var crcProviders = map[string]func(string) *crcValidator{
	"zoom": newZoomCRC,
}

// Config contains information about http_endpoint configuration
type config struct {
	Method                string                  `config:"method"`
	TLS                   *tlscommon.ServerConfig `config:"ssl"`
	BasicAuth             bool                    `config:"basic_auth"`
	Username              string                  `config:"username"`
	Password              string                  `config:"password"`
	ResponseCode          int                     `config:"response_code" validate:"positive"`
	ResponseBody          string                  `config:"response_body"`
	ListenAddress         string                  `config:"listen_address"`
	ListenPort            string                  `config:"listen_port"`
	URL                   string                  `config:"url" validate:"required"`
	Prefix                string                  `config:"prefix"`
	ContentType           string                  `config:"content_type"`
	Program               string                  `config:"program"`
	SecretHeader          string                  `config:"secret.header"`
	SecretValue           string                  `config:"secret.value"`
	HMACHeader            string                  `config:"hmac.header"`
	HMACKey               string                  `config:"hmac.key"`
	HMACType              string                  `config:"hmac.type"`
	HMACPrefix            string                  `config:"hmac.prefix"`
	CRCProvider           string                  `config:"crc.provider"`
	CRCSecret             string                  `config:"crc.secret"`
	IncludeHeaders        []string                `config:"include_headers"`
	PreserveOriginalEvent bool                    `config:"preserve_original_event"`
	Tracer                *lumberjack.Logger      `config:"tracer"`
}

func defaultConfig() config {
	return config{
		Method:        http.MethodPost,
		BasicAuth:     false,
		ResponseCode:  200,
		ResponseBody:  `{"message": "success"}`,
		ListenAddress: "127.0.0.1",
		ListenPort:    "8000",
		URL:           "/",
		Prefix:        "json",
		ContentType:   "application/json",
	}
}

func (c *config) Validate() error {
	if !json.Valid([]byte(c.ResponseBody)) {
		return errors.New("response_body must be valid JSON")
	}

	switch c.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
	default:
		return fmt.Errorf("method must be POST, PUT or PATCH: %s", c.Method)
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

	if c.CRCProvider != "" {
		if !isValidCRCProvider(c.CRCProvider) {
			return fmt.Errorf("not a valid CRC provider: %q", c.CRCProvider)
		} else if c.CRCSecret == "" {
			return errors.New("crc.secret is required when crc.provider is defined")
		}
	} else if c.CRCSecret != "" {
		return errors.New("crc.provider is required when crc.secret is defined")
	}

	return nil
}

func isValidCRCProvider(name string) bool {
	_, exists := crcProviders[strings.ToLower(name)]
	return exists
}

func canonicalizeHeaders(headerConf []string) (includeHeaders []string) {
	for i := range headerConf {
		headerConf[i] = textproto.CanonicalMIMEHeaderKey(headerConf[i])
	}
	return headerConf
}
