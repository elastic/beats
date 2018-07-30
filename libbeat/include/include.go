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

package include

import (
	"github.com/elastic/beats/libbeat/autodiscover/appenders/config"
	"github.com/elastic/beats/libbeat/autodiscover/providers/docker"
	"github.com/elastic/beats/libbeat/autodiscover/providers/jolokia"
	"github.com/elastic/beats/libbeat/autodiscover/providers/kubernetes"
	"github.com/elastic/beats/libbeat/feature"
	"github.com/elastic/beats/libbeat/outputs/codec/format"
	"github.com/elastic/beats/libbeat/outputs/codec/json"
	"github.com/elastic/beats/libbeat/outputs/console"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
	"github.com/elastic/beats/libbeat/outputs/fileout"
	"github.com/elastic/beats/libbeat/outputs/kafka"
	"github.com/elastic/beats/libbeat/outputs/logstash"
	"github.com/elastic/beats/libbeat/outputs/redis"
	"github.com/elastic/beats/libbeat/processors/actions"
	"github.com/elastic/beats/libbeat/processors/add_cloud_metadata"
	"github.com/elastic/beats/libbeat/processors/add_docker_metadata"
	"github.com/elastic/beats/libbeat/processors/add_host_metadata"
	"github.com/elastic/beats/libbeat/processors/add_kubernetes_metadata"
	"github.com/elastic/beats/libbeat/processors/add_locale"
	"github.com/elastic/beats/libbeat/processors/dissect"
	"github.com/elastic/beats/libbeat/publisher/queue/memqueue"
	"github.com/elastic/beats/libbeat/publisher/queue/spool"
)

// Bundle expose the main features.
var Bundle = feature.MustBundle(
	// Queues types
	feature.MustBundle(
		memqueue.Feature,
		spool.Feature,
	),

	// Autodiscovery providers
	feature.MustBundle(
		jolokia.Feature,
		docker.Feature,
		kubernetes.Feature,
	),

	// Autodiscovery appenders
	feature.MustBundle(
		config.Feature,
	),

	// Outputs
	feature.MustBundle(
		elasticsearch.Feature,
		logstash.Feature,
		redis.Feature,
		kafka.Feature,
		fileout.Feature,
		console.Feature,
	),
	// codecs
	feature.MustBundle(
		format.Feature,
		json.Feature,
	),

	// Processors
	feature.MustBundle(actions.Bundle,
		actions.Bundle,
		add_cloud_metadata.Feature,
		add_docker_metadata.Feature,
		add_host_metadata.Feature,
		add_kubernetes_metadata.Feature,
		add_locale.Feature,
		dissect.Feature,
	),
)

func init() {
	// Register main bundle
	feature.RegisterBundle(Bundle)
}
