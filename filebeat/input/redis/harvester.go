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
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	rd "github.com/gomodule/redigo/redis"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"

	"github.com/menderesk/beats/v7/filebeat/harvester"
)

// Harvester contains all redis harvester data
type Harvester struct {
	id        uuid.UUID
	done      chan struct{}
	conn      rd.Conn
	forwarder *harvester.Forwarder
}

// log contains all data related to one slowlog entry
//
// 	The data is in the following format:
// 	1) (integer) 13
// 	2) (integer) 1309448128
// 	3) (integer) 30
// 	4) 1) "slowlog"
// 	   2) "get"
// 	   3) "100"
//
type log struct {
	id        int64
	timestamp int64
	duration  int
	cmd       string
	key       string
	args      []string
}

// NewHarvester creates a new harvester with the given connection
func NewHarvester(conn rd.Conn) (*Harvester, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}

	return &Harvester{
		id:   id,
		done: make(chan struct{}),
		conn: conn,
	}, nil
}

// Run starts a new redis harvester
func (h *Harvester) Run() error {
	defer h.conn.Close()

	select {
	case <-h.done:
		return nil
	default:
	}
	// Writes Slowlog get and slowlog reset both to the buffer so they are executed together
	h.conn.Send("SLOWLOG", "GET")
	h.conn.Send("SLOWLOG", "RESET")

	// Flush the buffer to execute both commands and receive the reply from SLOWLOG GET
	h.conn.Flush()

	// Receives first reply from redis which is the one from GET
	logs, err := rd.Values(h.conn.Receive())
	if err != nil {
		return fmt.Errorf("error receiving slowlog data: %s", err)
	}

	// Read reply from RESET
	_, err = h.conn.Receive()
	if err != nil {
		return fmt.Errorf("error receiving reset data: %s", err)
	}

	for _, item := range logs {
		// Stopping here means some of the slowlog events are lost!
		select {
		case <-h.done:
			return nil
		default:
		}
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

		slowlogEntry := common.MapStr{
			"id":  log.id,
			"cmd": log.cmd,
			"key": log.key,
			"duration": common.MapStr{
				"us": log.duration,
			},
		}

		if log.args != nil {
			slowlogEntry["args"] = log.args
		}

		h.forwarder.Send(beat.Event{
			Timestamp: time.Unix(log.timestamp, 0).UTC(),
			Fields: common.MapStr{
				"message": strings.Join(args, " "),
				"redis": common.MapStr{
					"slowlog": slowlogEntry,
				},
				"event": common.MapStr{
					"created": time.Now(),
				},
			},
		})
	}
	return nil
}

// Stop stops the harvester
func (h *Harvester) Stop() {
	close(h.done)
}

// ID returns the unique harvester ID
func (h *Harvester) ID() uuid.UUID {
	return h.id
}
