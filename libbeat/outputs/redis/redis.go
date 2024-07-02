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
	"net/url"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/beats/v7/libbeat/outputs/outil"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/transport"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

const (
	defaultWaitRetry    = 1 * time.Second
	defaultMaxWaitRetry = 60 * time.Second
	defaultPort         = 6379
	redisScheme         = "redis"
	tlsRedisScheme      = "rediss"
)

func init() {
	outputs.RegisterType("redis", makeRedis)
}

func makeRedis(
	_ outputs.IndexManager,
	beat beat.Info,
	observer outputs.Observer,
	cfg *config.C,
) (outputs.Group, error) {

	if !cfg.HasField("index") {
		if err := cfg.SetString("index", -1, beat.Beat); err != nil {
			return outputs.Fail(err)
		}
	}

	err := cfgwarn.CheckRemoved6xSettings(cfg, "port")
	if err != nil {
		return outputs.Fail(err)
	}

	// ensure we have a `key` field in settings
	if !cfg.HasField("key") {
		s, err := cfg.String("index", -1)
		if err != nil {
			return outputs.Fail(err)
		}
		if err := cfg.SetString("key", -1, s); err != nil {
			return outputs.Fail(err)
		}
	}

	rConfig := defaultConfig
	if err := cfg.Unpack(&rConfig); err != nil {
		return outputs.Fail(err)
	}

	var dataType redisDataType
	switch rConfig.DataType {
	case "", "list":
		dataType = redisListType
	case "channel":
		dataType = redisChannelType
	default:
		return outputs.Fail(errors.New("Bad Redis data type"))
	}

	key, err := buildKeySelector(cfg)
	if err != nil {
		return outputs.Fail(err)
	}

	hosts, err := outputs.ReadHostList(cfg)
	if err != nil {
		return outputs.Fail(err)
	}

	tls, err := tlscommon.LoadTLSConfig(rConfig.TLS)
	if err != nil {
		return outputs.Fail(err)
	}

	clients := make([]outputs.NetworkClient, len(hosts))
	for i, h := range hosts {
		hasScheme := true
		if parts := strings.SplitN(h, "://", 2); len(parts) != 2 {
			h = fmt.Sprintf("%s://%s", redisScheme, h)
			hasScheme = false
		}

		hostUrl, err := url.Parse(h)
		if err != nil {
			return outputs.Fail(err)
		}

		if hostUrl.Host == "" {
			return outputs.Fail(fmt.Errorf("invalid redis url host %s", hostUrl.Host))
		}

		if hostUrl.Scheme != redisScheme && hostUrl.Scheme != tlsRedisScheme {
			return outputs.Fail(fmt.Errorf("invalid redis url scheme %s", hostUrl.Scheme))
		}

		transp := transport.Config{
			Timeout: rConfig.Timeout,
			Proxy:   &rConfig.Proxy,
			TLS:     tls,
			Stats:   observer,
		}

		switch hostUrl.Scheme {
		case redisScheme:
			if hasScheme {
				transp.TLS = nil // disable TLS if user explicitly set `redis` scheme
			}
		case tlsRedisScheme:
			if transp.TLS == nil {
				transp.TLS = &tlscommon.TLSConfig{} // enable with system default if TLS was not configured
			}
		}

		conn, err := transport.NewClient(transp, "tcp", hostUrl.Host, defaultPort)
		if err != nil {
			return outputs.Fail(err)
		}

		pass := rConfig.Password
		hostPass, passSet := hostUrl.User.Password()
		if passSet {
			pass = hostPass
		}

		enc, err := codec.CreateEncoder(beat, rConfig.Codec)
		if err != nil {
			return outputs.Fail(err)
		}

		client := newClient(conn, observer, rConfig.Timeout,
			pass, rConfig.Db, key, dataType, rConfig.Index, enc)
		clients[i] = newBackoffClient(client, rConfig.Backoff.Init, rConfig.Backoff.Max)
	}

	return outputs.SuccessNet(rConfig.Queue, rConfig.LoadBalance, rConfig.BulkMaxSize, 0, rConfig.MaxRetries, nil, clients)
}

func buildKeySelector(cfg *config.C) (outil.Selector, error) {
	return outil.BuildSelectorFromConfig(cfg, outil.Settings{
		Key:              "key",
		MultiKey:         "keys",
		EnableSingleOnly: true,
		FailEmpty:        true,
		Case:             outil.SelectorKeepCase,
	})
}
