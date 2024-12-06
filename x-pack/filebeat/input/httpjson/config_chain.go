// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Example:
// 1. First call to collect record ids
//   request_url: https://some_url.com/services/data/v1.0/records
//   response_json:
//     {
//         "records": [
//             {
//                 "id": 1,
//             },
//             {
//                 "id": 2,
//             },
//             {
//                 "id": 3,
//             },
//         ]
//     }
//
// 2. Second call to collect file name using collected ids from first call.
//   request_url using id as '1': https://some_url.com/services/data/v1.0/1/export_ids
//   response_json using id as '1':
//     {
//         "file_name": "file_1"
//     }
//   request_url using id as '2': https://some_url.com/services/data/v1.0/2/export_ids
//   response_json using id as '2':
//     {
//         "file_name": "file_2"
//     }
//
// 3. Third call to collect files using collected file names from second call.
//   request_url using file_name as 'file_1': https://some_url.com/services/data/v1.0/export_ids/file_1/info
//   request_url using file_name as 'file_2': https://some_url.com/services/data/v1.0/export_ids/file_2/info
//
//   Collect and make events from response in any format[csv, json, etc.] for all calls.
//
// Example configuration:
//
// - type: httpjson
//   enabled: true
//   request.url: https://some_url.com/services/data/v1.0/records (first call)
//   interval: 1h
//   chain:
//     - step:
//         request.url: https://some_url.com/services/data/v1.0/$.records[:].id/export_ids (second call)
//         request.method: GET
//         replace: $.records[:].id
//     - step:
//         request.url: https://some_url.com/services/data/v1.0/export_ids/$.file_name/info (third call)
//         request.method: GET
//         replace: $.file_name

package httpjson

// chainConfig for chain request.
// Following contains basic call structure for each call after normal httpjson
// call.
type chainConfig struct {
	Step  *stepConfig  `config:"step,omitempty"`
	While *whileConfig `config:"while,omitempty"`
}

// stepConfig will contain basic properties like, request.url,
// request.method and replace parameter. Each step: request.url
// will contain replace string with original URL to make a skeleton for the
// call request.
type stepConfig struct {
	Auth        *authConfig          `config:"auth"`
	Request     *requestConfig       `config:"request" validate:"required"`
	Response    *responseChainConfig `config:"response,omitempty"`
	Replace     string               `config:"replace,omitempty"`
	ReplaceWith string               `config:"replace_with,omitempty"`
}

// whileConfig will contain basic properties like auth parameters, request parameters,
// response parameters , a replace parameter and an expression parameter called 'until'.
// While is similar to stepConfig with the addition of 'until'. 'until' holds an expression
// and with the combination of "request.retry.max_attempts" retries a request 'until' the
// expression is evaluated to "true" or request.retry.max_attempts is exhausted. If
// request.retry.max_attempts is not specified , the max_attempts is always 1.
type whileConfig struct {
	Auth        *authConfig          `config:"auth"`
	Request     *requestConfig       `config:"request" validate:"required"`
	Response    *responseChainConfig `config:"response,omitempty"`
	Replace     string               `config:"replace,omitempty"`
	ReplaceWith string               `config:"replace_with,omitempty"`
	Until       *valueTpl            `config:"until" validate:"required"`
}

type responseChainConfig struct {
	Transforms transformsConfig `config:"transforms"`
	Split      *splitConfig     `config:"split"`
}

func defaultChainConfig() config {
	chaincfg := defaultConfig()
	chaincfg.Chain = []chainConfig{
		{
			While: &whileConfig{
				Auth:     chaincfg.Auth,
				Request:  chaincfg.Request,
				Response: &responseChainConfig{},
			},
			Step: &stepConfig{
				Auth:     chaincfg.Auth,
				Request:  chaincfg.Request,
				Response: &responseChainConfig{},
			},
		},
	}

	return chaincfg
}
