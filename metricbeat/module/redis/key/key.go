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

package key

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/menderesk/beats/v7/metricbeat/mb"
	"github.com/menderesk/beats/v7/metricbeat/mb/parse"
	"github.com/menderesk/beats/v7/metricbeat/module/redis"
)

var hostParser = parse.URLHostParserBuilder{DefaultScheme: "redis"}.Build()

func init() {
	mb.Registry.MustAddMetricSet("redis", "key", New,
		mb.WithHostParser(hostParser),
	)
}

// MetricSet for fetching Redis server information and statistics.
type MetricSet struct {
	*redis.MetricSet
	patterns []KeyPattern
}

// KeyPattern contains the information required to query keys
type KeyPattern struct {
	Keyspace *uint  `config:"keyspace"`
	Pattern  string `config:"pattern" validate:"required"`
	Limit    uint   `config:"limit"`
}

// New creates new instance of MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := struct {
		Patterns []KeyPattern `config:"key.patterns" validate:"nonzero,required"`
	}{}
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read configuration for 'key' metricset")
	}

	ms, err := redis.NewMetricSet(base)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create 'key' metricset")
	}

	return &MetricSet{
		MetricSet: ms,
		patterns:  config.Patterns,
	}, nil
}

// Fetch fetches information from Redis keys
func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	conn := m.Connection()
	defer func() {
		if err := conn.Close(); err != nil {
			m.Logger().Debug(errors.Wrapf(err, "failed to release connection"))
		}
	}()

	for _, p := range m.patterns {
		var keyspace uint
		if p.Keyspace == nil {
			keyspace = m.OriginalDBNumber()
		} else {
			keyspace = *p.Keyspace
		}
		if err := redis.Select(conn, keyspace); err != nil {
			msg := errors.Wrapf(err, "Failed to select keyspace %d", keyspace)
			m.Logger().Error(msg)
			r.Error(err)
			continue
		}

		keys, err := redis.FetchKeys(conn, p.Pattern, p.Limit)
		if err != nil {
			msg := errors.Wrapf(err, "Failed to list keys in keyspace %d with pattern '%s'", keyspace, p.Pattern)
			m.Logger().Error(msg)
			r.Error(err)
			continue
		}
		if p.Limit > 0 && len(keys) > int(p.Limit) {
			m.Logger().Debugf("Collecting stats for %d keys, but there are more available for pattern '%s' in keyspace %d", p.Limit)
			keys = keys[:p.Limit]
		}

		for _, key := range keys {
			keyInfo, err := redis.FetchKeyInfo(conn, key)
			if err != nil {
				msg := fmt.Errorf("Failed to fetch key info for key %s in keyspace %d", key, keyspace)
				m.Logger().Error(msg)
				r.Error(err)
				continue
			}
			if keyInfo == nil {
				m.Logger().Debugf("Ignoring removed key %s from keyspace %d", key, keyspace)
				continue
			}
			event := eventMapping(keyspace, keyInfo)
			if !r.Event(event) {
				m.Logger().Debug("Failed to report event, interrupting fetch")
				return nil
			}
		}
	}

	return nil
}
