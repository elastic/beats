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
	"encoding/json"
	"os"
	"strings"
	"time"

	rd "github.com/garyburd/redigo/redis"

	"github.com/elastic/beats/filebeat/channel"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/libbeat/logp"
)

func init() {
	err := input.Register("redis", NewInput)
	if err != nil {
		panic(err)
	}
}

// Input is a input for redis
type Input struct {
	started     bool
	outlet      channel.Outleter
	config      config
	cfg         *common.Config
	registry    *harvester.Registry
	passwordMap map[string]string
}

// loadPwdFile reads the redis password file and returns the password map
// sample file: https://github.com/oliver006/redis_exporter/blob/master/contrib/sample-pwd-file.json
func loadPwdFile(passwordFile string) (map[string]string, error) {
	res := make(map[string]string)

	logp.Debug("start load password file: %s", passwordFile)
	bytes, err := os.ReadFile(passwordFile)
	if err != nil {
		logp.Warn("load password file failed: %s", err)
		return nil, err
	}
	err = json.Unmarshal(bytes, &res)
	if err != nil {
		logp.Warn("password file format error: %s", err)
		return nil, err
	}

	logp.Info("Loaded %d entries from %s", len(res), passwordFile)
	for k := range res {
		logp.Debug("%s", k)
	}

	return res, nil
}

// NewInput creates a new redis input
func NewInput(cfg *common.Config, outletFactory channel.Connector, context input.Context) (input.Input, error) {
	cfgwarn.Experimental("Redis slowlog input is enabled.")

	config := defaultConfig

	err := cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}

	// 读取文件内容
	passwordMap := make(map[string]string)
	if config.PasswordFile != "" {
		passwordMap, err = loadPwdFile(config.PasswordFile)
		if err != nil {
			logp.Err("Error loading redis passwords from file %s, err: %s", config.PasswordFile, err)
		}
	}

	outlet, err := outletFactory(cfg, context.DynamicFields)
	if err != nil {
		return nil, err
	}

	p := &Input{
		started:     false,
		outlet:      outlet,
		config:      config,
		cfg:         cfg,
		registry:    harvester.NewRegistry(),
		passwordMap: passwordMap,
	}

	return p, nil
}

// LoadStates loads the states
func (p *Input) LoadStates(states []file.State) error {
	return nil
}

// Run runs the input
func (p *Input) Run() {
	logp.Debug("redis", "Run redis input with hosts: %+v", p.config.Hosts)

	if len(p.config.Hosts) == 0 {
		logp.Err("No redis hosts configured")
		return
	}

	forwarder := harvester.NewForwarder(p.outlet)
	for _, host := range p.config.Hosts {
		uri := host
		if !strings.Contains(uri, "://") {
			uri = "redis://" + uri
		}

		// 判断 password file 中是否存在该域名的密码配置，如果不存在，则使用默认密码
		password, ok := p.passwordMap[uri]
		if !ok {
			password = p.config.Password
		}

		pool := CreatePool(host, password, p.config.Network,
			p.config.MaxConn, p.config.IdleTimeout, p.config.IdleTimeout)

		h, err := NewHarvester(pool.Get())
		if err != nil {
			logp.Err("Failed to create harvester: %v", err)
			continue
		}
		h.forwarder = forwarder

		if err := p.registry.Start(h); err != nil {
			logp.Err("Harvester start failed: %s", err)
		}
	}
}

// Stop stops the input and all its harvesters
func (p *Input) Stop() {
	p.registry.Stop()
	p.outlet.Close()
}

// Reload runs the input
func (p *Input) Reload() {
	return
}

// Wait waits for the input to be completed. Not implemented.
func (p *Input) Wait() {}

// CreatePool creates a redis connection pool
// NOTE: This code is copied from the redis pool handling in metricbeat
func CreatePool(
	host, password, network string,
	maxConn int,
	idleTimeout, connTimeout time.Duration,
) *rd.Pool {
	return &rd.Pool{
		MaxIdle:     maxConn,
		IdleTimeout: idleTimeout,
		Dial: func() (rd.Conn, error) {
			c, err := rd.Dial(network, host,
				rd.DialConnectTimeout(connTimeout),
				rd.DialReadTimeout(connTimeout),
				rd.DialWriteTimeout(connTimeout))
			if err != nil {
				return nil, err
			}
			if password != "" {
				if _, err := c.Do("AUTH", password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
	}
}
