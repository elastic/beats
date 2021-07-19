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

package elasticsearch

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/libbeat/common/transport/kerberos"
)

type elasticsearchConfig struct {
	Protocol           string             `config:"protocol"`
	Path               string             `config:"path"`
	Params             map[string]string  `config:"parameters"`
	Headers            map[string]string  `config:"headers"`
	Username           string             `config:"username"`
	Password           string             `config:"password"`
	APIKey             string             `config:"api_key"`
	LoadBalance        bool               `config:"loadbalance"`
	CompressionLevel   int                `config:"compression_level" validate:"min=0, max=9"`
	EscapeHTML         bool               `config:"escape_html"`
	Kerberos           *kerberos.Config   `config:"kerberos"`
	BulkMaxSize        int                `config:"bulk_max_size"`
	MaxRetries         int                `config:"max_retries"`
	Backoff            Backoff            `config:"backoff"`
	NonIndexablePolicy NonIndexablePolicy `config:"non_indexable_policy"`

	Transport httpcommon.HTTPTransportSettings `config:",inline"`
}

type Backoff struct {
	Init time.Duration
	Max  time.Duration
}

type NonIndexablePolicy struct {
	Index  string
	Action string
}

const (
	defaultBulkSize = 50
)

var (
	defaultConfig = elasticsearchConfig{
		Protocol:         "",
		Path:             "",
		Params:           nil,
		Username:         "",
		Password:         "",
		APIKey:           "",
		MaxRetries:       3,
		CompressionLevel: 0,
		EscapeHTML:       false,
		Kerberos:         nil,
		LoadBalance:      true,
		Backoff: Backoff{
			Init: 1 * time.Second,
			Max:  60 * time.Second,
		},
		NonIndexablePolicy: NonIndexablePolicy{
			Action: "drop",
			Index:  "",
		},
		Transport: httpcommon.DefaultHTTPTransportSettings(),
	}
)

func (c *elasticsearchConfig) Validate() error {
	if c.APIKey != "" && (c.Username != "" || c.Password != "") {
		return fmt.Errorf("cannot set both api_key and username/password")
	}

	if !stringInSlice(c.NonIndexablePolicy.Action, []string{"drop", "death_letter_index"}) {
		return fmt.Errorf("invalid value for non_indexable_policy.action: %s, supported values are: drop, death_letter_index", c.NonIndexablePolicy.Action)
	}

	if c.NonIndexablePolicy.Action == "death_letter_index" && c.NonIndexablePolicy.Index == "" {
		return fmt.Errorf("empty or missing value for non_indexable_policy.index while 'non_indexable_policy.action: death_letter_index' is set ")
	}

	return nil
}

func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}
