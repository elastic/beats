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

import (
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
)

// chainConfig for chain request.
// Following contains basic call structure for each call after normal httpjson
// call.
type chainConfig struct {
	Step stepConfig `config:"step" validate:"required"`
}

// stepConfig will contain basic properties like, request.url,
// request.method and replace parameter. Each step: request.url
// will contain replace string with original URL to make a skeleton for the
// call request.
type stepConfig struct {
	Request  requestChainConfig  `config:"request"`
	Response responseChainConfig `config:"response,omitempty"`
	Replace  string              `config:"replace,omitempty"`
}

type requestChainConfig struct {
	URL        *urlConfig       `config:"url" validate:"required"`
	Method     string           `config:"method" validate:"required"`
	Body       *common.MapStr   `config:"body"`
	Transforms transformsConfig `config:"transforms"`

	Transport httpcommon.HTTPTransportSettings `config:",inline"`
}

type responseChainConfig struct {
	Transforms transformsConfig `config:"transforms"`
	Split      *splitConfig     `config:"split"`
}
