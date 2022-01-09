// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package instance

import (
	_ "github.com/elastic/beats/v7/libbeat/autodiscover/appenders/config" // Register autodiscover appenders
	_ "github.com/elastic/beats/v7/libbeat/autodiscover/providers/jolokia"
	_ "github.com/elastic/beats/v7/libbeat/monitoring/report/elasticsearch" // Register default monitoring reporting
	_ "github.com/elastic/beats/v7/libbeat/processors/actions"              // Register default processors.
	_ "github.com/elastic/beats/v7/libbeat/processors/add_cloud_metadata"
	_ "github.com/elastic/beats/v7/libbeat/processors/add_host_metadata"
	_ "github.com/elastic/beats/v7/libbeat/processors/add_id"
	_ "github.com/elastic/beats/v7/libbeat/processors/add_locale"
	_ "github.com/elastic/beats/v7/libbeat/processors/add_observer_metadata"
	_ "github.com/elastic/beats/v7/libbeat/processors/add_process_metadata"
	_ "github.com/elastic/beats/v7/libbeat/processors/communityid"
	_ "github.com/elastic/beats/v7/libbeat/processors/convert"
	_ "github.com/elastic/beats/v7/libbeat/processors/decode_xml"
	_ "github.com/elastic/beats/v7/libbeat/processors/decode_xml_wineventlog"
	_ "github.com/elastic/beats/v7/libbeat/processors/dissect"
	_ "github.com/elastic/beats/v7/libbeat/processors/dns"
	_ "github.com/elastic/beats/v7/libbeat/processors/extract_array"
	_ "github.com/elastic/beats/v7/libbeat/processors/fingerprint"
	_ "github.com/elastic/beats/v7/libbeat/processors/ratelimit"
	_ "github.com/elastic/beats/v7/libbeat/processors/registered_domain"
	_ "github.com/elastic/beats/v7/libbeat/processors/script"
	_ "github.com/elastic/beats/v7/libbeat/processors/translate_sid"
	_ "github.com/elastic/beats/v7/libbeat/processors/urldecode"
	_ "github.com/elastic/beats/v7/libbeat/publisher/includes" // Register publisher pipeline modules
)
