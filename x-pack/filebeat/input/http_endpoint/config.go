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

	"github.com/elastic/beats/v7/x-pack/filebeat/input/internal/httplog"
	"github.com/elastic/elastic-agent-libs/paths"
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
	OptionsHeaders        http.Header             `config:"options_headers"`
	OptionsStatus         int                     `config:"options_response_code"`
	ListenAddress         string                  `config:"listen_address"`
	ListenPort            string                  `config:"listen_port"`
	URL                   string                  `config:"url" validate:"required"`
	Prefix                string                  `config:"prefix"`
	ContentType           string                  `config:"content_type"`
	MaxBodySize           *int64                  `config:"max_body_bytes"`
	MaxInFlight           int64                   `config:"max_in_flight_bytes"`
	HighWaterInFlight     int64                   `config:"high_water_in_flight_bytes"`
	LowWaterInFlight      int64                   `config:"low_water_in_flight_bytes"`
	RetryAfter            int                     `config:"retry_after"`
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
	Tracer                *tracerConfig           `config:"tracer"`
}

type tracerConfig struct {
	Enabled           *bool `config:"enabled"`
	lumberjack.Logger `config:",inline"`
}

func (t *tracerConfig) enabled() bool {
	return t != nil && (t.Enabled == nil || *t.Enabled)
}

func defaultConfig() config {
	return config{
		Method:        http.MethodPost,
		BasicAuth:     false,
		ResponseCode:  200,
		ResponseBody:  `{"message": "success"}`,
		OptionsStatus: 200,
		RetryAfter:    10,
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

	if c.MaxBodySize != nil && *c.MaxBodySize < 0 {
		return fmt.Errorf("max_body_bytes is negative: %d", *c.MaxBodySize)
	}

	// Apply defaults for in-flight byte limits and validate their relationships.
	c.applyInFlightDefaults()
	if err := c.validateInFlightLimits(); err != nil {
		return err
	}

	if !c.Tracer.enabled() {
		return nil
	}
	if c.Tracer.Filename == "" {
		return errors.New("request tracer must have a filename if used")
	}
	if c.Tracer.MaxSize == 0 {
		// By default Lumberjack caps file sizes at 100MB which
		// is excessive for a debugging logger, so default to 1MB
		// which is the minimum.
		c.Tracer.MaxSize = 1
	}
	ok, err := httplog.IsPathInLogsFor(inputName, c.Tracer.Filename)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("request tracer path must be within %q path", paths.Resolve(paths.Logs, inputName))
	}

	return nil
}

// applyInFlightDefaults sets default values for high_water_in_flight_bytes and
// low_water_in_flight_bytes based on max_in_flight_bytes if they are not explicitly set.
func (c *config) applyInFlightDefaults() {
	if c.MaxInFlight <= 0 {
		return
	}
	if c.HighWaterInFlight == 0 {
		// Default high water is half of maximum in flight.
		// This is conservative.
		c.HighWaterInFlight = c.MaxInFlight / 2
	}
	if c.LowWaterInFlight == 0 {
		const kB = 1 << 10
		// Low water is the lesser of 80% of high water or high water less 64kB clamped non-negative.
		c.LowWaterInFlight = min(c.HighWaterInFlight*4/5, max(0, c.HighWaterInFlight-64*kB))
	}
}

// validateInFlightLimits validates the relationships between the in-flight byte limits.
func (c *config) validateInFlightLimits() error {
	if c.MaxInFlight < 0 {
		return fmt.Errorf("max_in_flight_bytes is negative: %d", c.MaxInFlight)
	}
	if c.HighWaterInFlight < 0 {
		return fmt.Errorf("high_water_in_flight_bytes is negative: %d", c.HighWaterInFlight)
	}
	if c.LowWaterInFlight < 0 {
		return fmt.Errorf("low_water_in_flight_bytes is negative: %d", c.LowWaterInFlight)
	}
	if c.MaxInFlight == 0 && (c.HighWaterInFlight != 0 || c.LowWaterInFlight != 0) {
		return errors.New("high_water_in_flight_bytes and low_water_in_flight_bytes require max_in_flight_bytes to be set")
	}
	if c.MaxInFlight > 0 {
		if c.MaxInFlight < 2 {
			return fmt.Errorf("max_in_flight_bytes must be at least 2: currently set to %d", c.MaxInFlight)
		}
		if c.HighWaterInFlight >= c.MaxInFlight {
			return fmt.Errorf("high_water_in_flight_bytes (%d) must be less than max_in_flight_bytes (%d)", c.HighWaterInFlight, c.MaxInFlight)
		}
		if c.LowWaterInFlight >= c.HighWaterInFlight {
			return fmt.Errorf("low_water_in_flight_bytes (%d) must be less than high_water_in_flight_bytes (%d)", c.LowWaterInFlight, c.HighWaterInFlight)
		}
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
