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

package ecs

// The data_stream fields take part in defining the new data stream naming
// scheme.
// In the new data stream naming scheme the value of the data stream fields
// combine to the name of the actual data stream in the following manner:
// `{data_stream.type}-{data_stream.dataset}-{data_stream.namespace}`. This
// means the fields can only contain characters that are valid as part of names
// of data streams. More details about this can be found in this
// https://www.elastic.co/blog/an-introduction-to-the-elastic-data-stream-naming-scheme[blog
// post].
// An Elasticsearch data stream consists of one or more backing indices, and a
// data stream name forms part of the backing indices names. Due to this
// convention, data streams must also follow index naming restrictions. For
// example, data stream names cannot include `\`, `/`, `*`, `?`, `"`, `<`, `>`,
// `|`, ` ` (space character), `,`, or `#`. Please see the Elasticsearch
// reference for additional
// https://www.elastic.co/guide/en/elasticsearch/reference/current/indices-create-index.html#indices-create-api-path-params[restrictions].
type DataStream struct {
	// An overarching type for the data stream.
	// Currently allowed values are "logs" and "metrics". We expect to also add
	// "traces" and "synthetics" in the near future.
	Type string `ecs:"type"`

	// The field can contain anything that makes sense to signify the source of
	// the data.
	// Examples include `nginx.access`, `prometheus`, `endpoint` etc. For data
	// streams that otherwise fit, but that do not have dataset set we use the
	// value "generic" for the dataset value. `event.dataset` should have the
	// same value as `data_stream.dataset`.
	// Beyond the Elasticsearch data stream naming criteria noted above, the
	// `dataset` value has additional restrictions:
	//   * Must not contain `-`
	//   * No longer than 100 characters
	Dataset string `ecs:"dataset"`

	// A user defined namespace. Namespaces are useful to allow grouping of
	// data.
	// Many users already organize their indices this way, and the data stream
	// naming scheme now provides this best practice as a default. Many users
	// will populate this field with `default`. If no value is used, it falls
	// back to `default`.
	// Beyond the Elasticsearch index naming criteria noted above, `namespace`
	// value has the additional restrictions:
	//   * Must not contain `-`
	//   * No longer than 100 characters
	Namespace string `ecs:"namespace"`
}
