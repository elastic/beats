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

package slowlog

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"
	"github.com/elastic/beats/metricbeat/module/redis"

	rd "github.com/garyburd/redigo/redis"
)

var (
	debugf = logp.MakeDebug("redis-slowlog")
)

// log contains all data related to one slowlog entry
type log struct {
	id        int64    // A unique progressive identifier for every slow log entry.
	timestamp int64    // The unix timestamp at which the logged command was processed.
	duration  int      // The amount of time needed for its execution, in microseconds.
	cmd       string   // The array composing the arguments of the command.
	key       string   // Client IP address and port (4.0 only).
	args      []string // Client name if set via the CLIENT SETNAME command (4.0 only).
}

func init() {
	mb.Registry.MustAddMetricSet("redis", "slowlog", New,
		mb.WithHostParser(parse.PassThruHostParser),
		mb.DefaultMetricSet(),
	)
}

// MetricSet for fetching Redis slowlogs.
type MetricSet struct {
	mb.BaseMetricSet
	pool *rd.Pool
}

// New creates new instance of MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	// Unpack additional configuration options.
	config := struct {
		IdleTimeout time.Duration `config:"idle_timeout"`
		Network     string        `config:"network"`
		MaxConn     int           `config:"maxconn" validate:"min=1"`
		Password    string        `config:"password"`
	}{
		Network:  "tcp",
		MaxConn:  10,
		Password: "",
	}
	err := base.Module().UnpackConfig(&config)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		pool: redis.CreatePool(base.Host(), config.Password, config.Network,
			config.MaxConn, config.IdleTimeout, base.Module().Config().Timeout),
	}, nil
}

// Fetch fetches metrics from Redis by issuing the SLOWLOG command.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	// Issue slowlog len.
	count, err := redis.FetchSlowLogCount(m.pool.Get())
	if err != nil {
		return nil, err
	}

	// Issue slowlog get
	slowlogs, err := redis.FetchSlowLog(m.pool.Get())
	if err != nil {
		return nil, err
	}

	var events []common.MapStr
	for _, item := range slowlogs {
		entry, err := rd.Values(item, nil)
		if err != nil {
			logp.Err("Error loading slowlog values: %s", err)
			continue
		}

		var log log
		var args []string
		rd.Scan(entry, &log.id, &log.timestamp, &log.duration, &args)

		// This splits up the args into cmd, key, args.
		argsLen := len(args)
		if argsLen > 0 {
			log.cmd = args[0]
		}
		if argsLen > 1 {
			log.key = args[1]
		}

		// This could contain confidential data, processors should be used to drop it if needed
		if argsLen > 2 {
			log.args = args[2:]
		}

		event := common.MapStr{
			"id":  log.id,
			"cmd": log.cmd,
			"key": log.key,
			"duration": common.MapStr{
				"us": log.duration,
			},
		}

		if log.args != nil {
			event["args"] = log.args
		}

		events = append(events, event)
	}

	debugf("Redis SLOWLOG from %s: %+v", m.Host(), events)
	return events, nil
}
