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

package redis

import (
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/codec"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type redisConfig struct {
	Password    string                `config:"password"`
	Index       string                `config:"index"`
	Key         string                `config:"key"`
	Port        int                   `config:"port"`
	LoadBalance bool                  `config:"loadbalance"`
	Timeout     time.Duration         `config:"timeout"`
	BulkMaxSize int                   `config:"bulk_max_size"`
	MaxRetries  int                   `config:"max_retries"`
	TLS         *tlscommon.Config     `config:"ssl"`
	Proxy       transport.ProxyConfig `config:",inline"`
	Codec       codec.Config          `config:"codec"`
	Db          int                   `config:"db"`
	DataType    string                `config:"datatype"`
}

var (
	defaultConfig = redisConfig{
		Port:        6379,
		LoadBalance: true,
		Timeout:     5 * time.Second,
		BulkMaxSize: 2048,
		MaxRetries:  3,
		TLS:         nil,
		Db:          0,
		DataType:    "list",
	}
)

func (c *redisConfig) Validate() error {
	switch c.DataType {
	case "", "list", "channel":
	default:
		return fmt.Errorf("redis data type %v not supported", c.DataType)
	}

	if c.Key != "" && c.Index != "" {
		return errors.New("Cannot use both `output.redis.key` and `output.redis.index` configuration options." +
			" Set only `output.redis.key`")
	}

	if c.Key == "" && c.Index != "" {
		c.Key = c.Index
		logp.Warn("The `output.redis.index` configuration setting is deprecated. Use `output.redis.key` instead.")
	}

	return nil
}
