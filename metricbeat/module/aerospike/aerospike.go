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

package aerospike

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"

	as "github.com/aerospike/aerospike-client-go"
)

type Config struct {
	ClusterName string            `config:"cluster_name"`
	TLS         *tlscommon.Config `config:"ssl"`
}

// DefaultConfig return default config for the aerospike module.
func DefaultConfig() Config {
	return Config{}
}

func ParseClientPolicy(config Config) (*as.ClientPolicy, error) {
	clientPolicy := as.NewClientPolicy()
	if config.TLS.IsEnabled() {
		tlsconfig, err := tlscommon.LoadTLSConfig(config.TLS)
		if err != nil {
			return nil, fmt.Errorf("could not initialize TLS configurations %w", err)
		}
		clientPolicy.TlsConfig = tlsconfig.ToConfig()
	}

	if config.ClusterName != "" {
		clientPolicy.ClusterName = config.ClusterName
	}
	return clientPolicy, nil
}

func ParseHost(host string) (*as.Host, error) {
	pieces := strings.Split(host, ":")
	if len(pieces) != 2 {
		return nil, fmt.Errorf("Can't parse host %s", host)
	}
	port, err := strconv.Atoi(pieces[1])
	if err != nil {
		return nil, fmt.Errorf("Can't parse port: %w", err)
	}
	return as.NewHost(pieces[0], port), nil
}

func ParseInfo(info string) map[string]interface{} {
	result := make(map[string]interface{})

	for _, keyValueStr := range strings.Split(info, ";") {
		KeyValArr := strings.Split(keyValueStr, "=")
		if len(KeyValArr) == 2 {
			result[KeyValArr[0]] = KeyValArr[1]
		}
	}

	return result
}
