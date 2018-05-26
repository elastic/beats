package sip

import (
    "github.com/elastic/beats/packetbeat/config"
)

type sipConfig struct {
    config.ProtocolCommon           `config:",inline"`
    IncludeRawMessage     bool      `config:"include_raw"`
    IncludeHeaders        bool      `config:"include_headers"`
    IncludeBody           bool      `config:"include_body"`
    ParseDetail           bool      `config:"parse_detail"`
    UseDefaultHeaders     bool      `config:"use_default_headers"`
    HeadersToParseAsURI   []string  `config:"parse_as_uri_for"`
    HeadersToParseAsInt   []string  `config:"parse_as_int_for"`
}

var (
    defaultConfig = sipConfig{
        ProtocolCommon: config.ProtocolCommon{},
        IncludeRawMessage: true,
        IncludeHeaders: true,
        IncludeBody  : true,
		ParseDetail: false,
		UseDefaultHeaders: true,
        HeadersToParseAsURI: []string{},
        HeadersToParseAsInt: []string{},
    }
)

