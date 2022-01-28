// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package httpjson

// chainConfig for chain request.
// Following contains basic call structure for each call after normal httpjson
// call. each call will be defined in step configuration.
// each call can use one of the key/values from the previous call to manipulate
// the new URL. each step configuration will contain basic properties
// request.url, request.method and replace parameter. each step: request.url
// will contain replace string with original URL to make a skeleton for the
// call request.
//
// example configuration:
//
// chain:
//     - step:
//         request.url: https://some_url.com/services/data/v1.0/records.#.Id/export_ids
//         request.method: GET
//         replace: records.#.Id
//     - step:
//         request.url: https://some_url.com/services/data/v1.0/export_ids/export.data.#.Id/info
//         request.method: GET
//         replace: export.data.#.Id
//
// Normal call:
//
// the normal call which is the first call in httpjson will collect JSON data.
//
// Example JSON:
//
// 	{
// 		"token": "ahskfdheuicjksah",
// 		"records": [
// 			{
// 				"Id": "id-1",
// 				"path": "12kjkhksad123/"
// 			},
// 			{
// 				"Id": "id-2",
// 				"path": "12kjkhksad123/"
// 			}
// 		]
// 	}
//
// step-1 call:
//
// step-1 call which is the first call in chain httpjson will collect JSON data.
//
// step-1 will use `replace` to parse and collect data from collected JSON data from a normal call.
//
// Collected ids from JSON Data:
// [ "id-1", "id-2" ]
//
// These collected ids will be used to construct a new URL.
//
// Request-url-1: https://some_url.com/services/data/v1.0/id-1/export_ids
//
// Response:
//
// 	{
// 		"token": "ahskfdheuicjksah",
// 		"export": {
// 			"data": [
// 				{
// 					"Id": "export_id-1"
// 				},
// 				{
// 					"Id": "export_id-2"
// 				}
// 			]
// 		}
// 	}
//
// request-url-1: https://some_url.com/services/data/v1.0/id-2/export_ids
//
// response:
//
// 	{
// 		"token": "ahskfdheuicjksah",
// 		"export": {
// 			"data": [
// 				{
// 					"Id": "export_id-3"
// 				},
// 				{
// 					"Id": "export_id-4"
// 				}
// 			]
// 		}
// 	}
//
// step-2 call:
//
// the step-2 call which is the second call in chain httpjson will collect JSON data.
//
// step-2 will use `replace` to parse and collect data from collected JSON data from a step-1 call.
//
// collcted ids from JSON Data:
// [ "export_id-1", "export_id-2", "export_id-3", "export_id-4" ]
//
// These collected ids will be used to construct a new URL.
//
// request-url-1: https://some_url.com/services/data/v1.0/export_ids/export_id-1/info
//
// request-url-1: https://some_url.com/services/data/v1.0/export_ids/export_id-2/info
//
// request-url-1: https://some_url.com/services/data/v1.0/export_ids/export_id-3/info
//
// request-url-1: https://some_url.com/services/data/v1.0/export_ids/export_id-4/info
//
// Collect all the responses from newly constructed URLs.
//
// Response:
//
// 	{
// 		"token": "ahskfdheuicjksah",
// 		"exports": {
// 			"data": {
// 				"some_data": "data",
// 			}
// 		}
// 	}
type chainConfig struct {
	Step stepConfig `config:"step" validate:"required"`
}

type stepConfig struct {
	Auth     authConfig     `config:"auth,omitempty"`
	Request  requestConfig  `config:"request"`
	Response responseConfig `config:"response,omitempty"`
	Replace  string         `config:"replace,omitempty"`
}
