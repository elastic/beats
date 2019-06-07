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

package elb

import (
	"github.com/elastic/beats/libbeat/autodiscover/template"
	"github.com/elastic/beats/libbeat/common"
)

// Config for the aws_elb autodiscover provider.
type Config struct {
	Region string `config:"region" validate:"required"`
	Type   string `config:"type"`
	// Hints are currently not supported, but may be implemented in a later release
	HintsEnabled bool                    `config:"hints.enabled"`
	Builders     []*common.Config        `config:"builders"`
	Appenders    []*common.Config        `config:"appenders"`
	Templates    template.MapperSettings `config:"templates"`
	Credentials  *AWSCredentials         `config:credentials`
}

type AWSCredentials struct {
	AccessKeyID     string `config:"access_key_id"`
	SecretAccessKey string `config:"secret_access_key"`
}

func defaultConfig() *Config {
	return &Config{}
}
