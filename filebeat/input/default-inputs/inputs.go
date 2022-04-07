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

package inputs

import (
	"github.com/elastic/beats/v8/filebeat/beater"
	"github.com/elastic/beats/v8/filebeat/input/filestream"
	"github.com/elastic/beats/v8/filebeat/input/kafka"
	"github.com/elastic/beats/v8/filebeat/input/unix"
	v2 "github.com/elastic/beats/v8/filebeat/input/v2"
	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/logp"
)

func Init(info beat.Info, log *logp.Logger, components beater.StateStore) []v2.Plugin {
	return append(
		genericInputs(log, components),
		osInputs(info, log, components)...,
	)
}

func genericInputs(log *logp.Logger, components beater.StateStore) []v2.Plugin {
	return []v2.Plugin{
		filestream.Plugin(log, components),
		kafka.Plugin(),
		unix.Plugin(),
	}
}
