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

package mongodb

import (
	"crypto/tls"
	"net"

	"gopkg.in/mgo.v2"

	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
)

// ModuleConfig contains the common configuration for this module
type ModuleConfig struct {
	TLS *tlscommon.Config `config:"ssl"`
}

// MetricSet type defines all fields of the MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	DialInfo *mgo.DialInfo
}

// NewMetricSet creates a new instance of the MetricSet
func NewMetricSet(base mb.BaseMetricSet) (*MetricSet, error) {
	var config ModuleConfig
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	dialInfo, err := mgo.ParseURL(base.HostData().URI)
	if err != nil {
		return nil, err
	}
	dialInfo.Timeout = base.Module().Config().Timeout

	if config.TLS.IsEnabled() {
		tlsConfig, err := tlscommon.LoadTLSConfig(config.TLS)
		if err != nil {
			return nil, err
		}

		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			hostname, _, err := net.SplitHostPort(base.HostData().Host)
			if err != nil {
				logp.Warn("Failed to obtain hostname from `%s`: %s", hostname, err)
				hostname = ""
			}
			return tls.Dial("tcp", addr.String(), tlsConfig.BuildModuleConfig(hostname))
		}
	}

	return &MetricSet{
		BaseMetricSet: base,
		DialInfo:      dialInfo,
	}, nil
}
