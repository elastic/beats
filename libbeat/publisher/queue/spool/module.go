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

package spool

import (
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/paths"
	"github.com/elastic/beats/v7/libbeat/publisher/queue"
	"github.com/elastic/go-txfile"
)

func init() {
	queue.RegisterQueueType(
		"spool",
		create,
		feature.MakeDetails(
			"Disk spool",
			"Buffer events in disk spool before sending to the output.",
			feature.Beta))
}

func create(
	ackListener queue.ACKListener, logp *logp.Logger, cfg *common.Config, inQueueSize int,
) (queue.Queue, error) {
	cfgwarn.Beta("Spooling to disk is beta")

	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	path := config.File.Path
	if path == "" {
		path = paths.Resolve(paths.Data, "spool.dat")
	}

	flushEvents := uint(0)
	if count := config.Write.FlushEvents; count > 0 {
		flushEvents = uint(count)
	}

	var log logger = logp
	if logp == nil {
		log = defaultLogger()
	}

	return newDiskSpool(log, path, settings{
		ACKListener:       ackListener,
		Mode:              config.File.Permissions,
		WriteBuffer:       uint(config.Write.BufferSize),
		WriteFlushTimeout: config.Write.FlushTimeout,
		WriteFlushEvents:  flushEvents,
		ReadFlushTimeout:  config.Read.FlushTimeout,
		Codec:             config.Write.Codec,
		File: txfile.Options{
			MaxSize:  uint64(config.File.MaxSize),
			PageSize: uint32(config.File.PageSize),
			Prealloc: config.File.Prealloc,
			Readonly: false,
		},
	})
}
